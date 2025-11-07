package server

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"

	"github.com/firefart/go-webserver-template/internal/cacher"
	"github.com/firefart/go-webserver-template/internal/config"
	"github.com/firefart/go-webserver-template/internal/database"
	inthttp "github.com/firefart/go-webserver-template/internal/http"
	"github.com/firefart/go-webserver-template/internal/mail"
	"github.com/firefart/go-webserver-template/internal/metrics"
	"github.com/firefart/go-webserver-template/internal/server/handlers"
	"github.com/firefart/go-webserver-template/internal/server/httperror"
	"github.com/firefart/go-webserver-template/internal/server/middleware"
	"github.com/firefart/go-webserver-template/internal/server/router"
	"github.com/nikoksr/notify"
)

type server struct {
	logger     *slog.Logger
	config     config.Configuration
	db         database.Interface
	notify     *notify.Notify
	metrics    *metrics.Metrics
	cache      *cacher.Cache[string]
	mailer     mail.Interface
	httpClient *inthttp.Client
	accessLog  bool
	debug      bool
}

//go:embed assets
var fsAssets embed.FS

func notFound(w http.ResponseWriter, _ *http.Request) error {
	content, err := fs.ReadFile(fsAssets, "assets/error_pages/HTTP404.html")
	if err != nil {
		return err
	}
	http.Error(w, string(content), http.StatusNotFound)
	return nil
}

func staticFile(w http.ResponseWriter, r *http.Request, filesystem fs.FS, path string) error {
	f, err := filesystem.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open %s", path)
	}
	defer f.Close()
	stat, err := f.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat %s", path)
	}
	ff, ok := f.(io.ReadSeeker)
	if !ok {
		return fmt.Errorf("file %s does not implement io.ReadSeeker", path)
	}
	http.ServeContent(w, r, path, stat.ModTime(), ff)
	return nil
}

func NewServer(opts ...OptionsServerFunc) (http.Handler, error) {
	s := server{
		logger: slog.New(slog.DiscardHandler),
		debug:  false,
	}

	for _, o := range opts {
		if err := o(&s); err != nil {
			return nil, err
		}
	}

	r := router.New()

	r.SetErrorHandler(func(w http.ResponseWriter, r *http.Request, err error) {
		s.metrics.Errors.WithLabelValues(r.Host).Inc()
		s.logger.Error("error on request", slog.String("err", err.Error()))
		var httpErr *httperror.HTTPError
		code := http.StatusInternalServerError
		if errors.As(err, &httpErr) {
			code = httpErr.StatusCode
		}

		// send an asynchronous notification (but ignore 404 and stuff)
		if err != nil && code > 499 {
			s.logger.Error("error on request", slog.String("err", err.Error()))

			go func(e error) {
				s.logger.Debug("sending error notification", slog.String("err", e.Error()))
				if err2 := s.notify.Send(context.Background(), "ERROR", e.Error()); err2 != nil {
					s.logger.Error("error on notification send", slog.String("err", err2.Error()))
				}
			}(err)
		}

		// send error page
		errorPage := fmt.Sprintf("assets/error_pages/HTTP%d.html", code)
		if _, err := fs.Stat(fsAssets, errorPage); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				errorPage = "assets/error_pages/HTTP500.html"
			} else {
				s.logger.Error("could not check if file exists", slog.String("err", err.Error()))
				errorPage = "assets/error_pages/HTTP500.html"
			}
		}

		content, err2 := fsAssets.ReadFile(errorPage)
		if err2 != nil {
			s.logger.Error("could not read error page", slog.String("err", err2.Error()))
			return
		}
		http.Error(w, string(content), code)
	})

	r.Use(middleware.Recover(s.logger))
	r.Use(middleware.RealIP(middleware.RealIPConfig{
		IPHeader: s.config.Server.IPHeader,
	}))
	r.Use(middleware.RealHost(middleware.RealHostConfig{
		Headers: s.config.Server.HostHeaders,
	}))
	if s.accessLog {
		r.Use(middleware.AccessLog(middleware.AccessLogConfig{
			Logger:  s.logger,
			Metrics: s.metrics,
		}))
	}

	static, err := fs.Sub(fsAssets, "assets/web")
	if err != nil {
		return nil, err
	}

	r.HandleFunc("/robots.txt", func(w http.ResponseWriter, r *http.Request) error {
		return staticFile(w, r, static, "robots.txt")
	})
	scripts, err := fs.Sub(static, "scripts")
	if err != nil {
		return nil, err
	}
	r.Handle("/scripts", http.FileServerFS(scripts))

	css, err := fs.Sub(static, "css")
	if err != nil {
		return nil, err
	}
	r.Handle("/css", http.FileServerFS(css))

	r.HandleFunc("/", handlers.NewIndexHandler(s.debug).Handler)

	r.Group(func(r *router.Router) {
		r.Use(middleware.SecretKeyHeader(middleware.SecretKeyHeaderConfig{
			SecretKeyHeaderName:  s.config.Server.SecretKeyHeaderName,
			SecretKeyHeaderValue: s.config.Server.SecretKeyHeaderValue,
			Logger:               s.logger,
			Debug:                s.debug,
		}))

		// health check for monitoring
		r.HandleFunc(fmt.Sprintf("GET %s", "/health"), handlers.NewHealthHandler().Handler)
		r.HandleFunc(fmt.Sprintf("GET %s", "/version"), handlers.NewVersionHandler().Handler)
	})

	// custom 404 for the rest
	r.HandleFunc("/", notFound)

	return r, nil
}
