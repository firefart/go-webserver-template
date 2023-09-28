package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"html/template"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/patrickmn/go-cache"
	"github.com/sirupsen/logrus"
)

//go:embed templates
//go:embed assets
var staticFS embed.FS

type application struct {
	logger   *logrus.Logger
	debug    bool
	config   Configuration
	renderer *TemplateRenderer
	cache    *cache.Cache
}

// TemplateRenderer is a custom html/template renderer for Echo framework
type TemplateRenderer struct {
	templates *template.Template
}

// Render renders a template document
func (t *TemplateRenderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

func main() {
	logger := logrus.New()
	logger.SetOutput(os.Stdout)
	logger.SetLevel(logrus.InfoLevel)
	if err := run(logger); err != nil {
		logger.Errorf("[ERROR] %s", err)
	}
}

func run(logger *logrus.Logger) error {
	app := &application{
		logger: logger,
	}

	var configFile string
	var debugOutput bool
	flag.StringVar(&configFile, "c", "", "config file to use")
	flag.BoolVar(&debugOutput, "debug", false, "enable debug logging")
	flag.Parse()

	if configFile == "" {
		return fmt.Errorf("please provide a config file")
	}

	config, err := GetConfig(configFile)
	if err != nil {
		return err
	}
	app.config = config

	if debugOutput {
		app.logger.SetLevel(logrus.DebugLevel)
		app.debug = true
	}

	app.logger.Info("Starting server with the following parameters:")
	app.logger.Infof("port: %d", config.Server.Port)
	app.logger.Infof("graceful timeout: %s", config.Server.GracefulTimeout)
	app.logger.Infof("timeout: %s", config.Timeout)
	app.logger.Infof("debug: %t", app.debug)

	app.renderer = &TemplateRenderer{
		templates: template.Must(template.New("").Funcs(template.FuncMap{"StringsJoin": strings.Join}).ParseFS(staticFS, "templates/*")),
	}
	app.cache = cache.New(config.Cache.Timeout, config.Cache.Timeout)

	srv := &http.Server{
		Addr:    net.JoinHostPort("", strconv.Itoa(config.Server.Port)),
		Handler: app.routes(),
	}

	go func() {
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			app.logger.Error(err)
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGTERM, syscall.SIGINT)
	<-c
	ctx, cancel := context.WithTimeout(context.Background(), config.Server.GracefulTimeout)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		app.logger.Error(err)
	}
	app.logger.Info("shutting down")
	os.Exit(0)
	return nil
}

func (app *application) routes() http.Handler {
	e := echo.New()
	e.HideBanner = true
	e.Debug = app.debug
	e.Renderer = app.renderer

	if app.config.Cloudflare {
		e.IPExtractor = echo.ExtractIPFromRealIPHeader()
	}

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.StaticFS("/static", echo.MustSubFS(staticFS, "assets"))
	e.GET("/", func(c echo.Context) error {
		return c.Render(http.StatusOK, "index.html", nil)
	})
	e.GET("/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "route")
	})
	return e
}
