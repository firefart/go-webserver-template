package main

import (
	"context"
	"embed"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/lmittmann/tint"
	"github.com/mattn/go-isatty"
	"github.com/nikoksr/notify"
	"github.com/nikoksr/notify/service/discord"
	"github.com/nikoksr/notify/service/mail"
	"github.com/nikoksr/notify/service/msteams"
	"github.com/nikoksr/notify/service/sendgrid"
	"github.com/nikoksr/notify/service/telegram"
	"github.com/patrickmn/go-cache"

	_ "net/http/pprof"

	_ "go.uber.org/automaxprocs"
)

//go:embed templates
//go:embed assets
var staticFS embed.FS

//go:embed error_pages
var errorPages embed.FS

var secretKeyHeaderName = http.CanonicalHeaderKey("X-Secret-Key-Header")
var cloudflareIPHeaderName = http.CanonicalHeaderKey("CF-Connecting-IP")

type application struct {
	logger   *slog.Logger
	debug    bool
	config   Configuration
	renderer *TemplateRenderer
	cache    *cache.Cache
	notify   *notify.Notify
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
	var debugMode bool
	var configFilename string
	var jsonOutput bool
	flag.BoolVar(&debugMode, "debug", false, "Enable DEBUG mode")
	flag.StringVar(&configFilename, "config", "", "config file to use")
	flag.BoolVar(&jsonOutput, "json", false, "output in json instead")
	flag.Parse()

	w := os.Stdout
	var level = new(slog.LevelVar)
	level.Set(slog.LevelInfo)

	var replaceFunc func(groups []string, a slog.Attr) slog.Attr = nil
	if debugMode {
		level.Set(slog.LevelDebug)
		// add source file information
		wd, err := os.Getwd()
		if err != nil {
			panic("unable to determine working directory")
		}
		replaceFunc = func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.SourceKey {
				source := a.Value.Any().(*slog.Source)
				// remove current working directory and only leave the relative path to the program
				if file, ok := strings.CutPrefix(source.File, wd); ok {
					source.File = file
				}
			}
			return a
		}
	}

	var handler slog.Handler
	if jsonOutput {
		handler = slog.NewJSONHandler(w, &slog.HandlerOptions{
			Level:       level,
			AddSource:   debugMode,
			ReplaceAttr: replaceFunc,
		})
	} else {
		textOptions := &tint.Options{
			Level:       level,
			NoColor:     !isatty.IsTerminal(w.Fd()),
			AddSource:   debugMode,
			ReplaceAttr: replaceFunc,
		}
		handler = tint.NewHandler(w, textOptions)
	}

	logger := slog.New(handler)

	if err := run(logger, configFilename, debugMode); err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
}

func run(logger *slog.Logger, configFilename string, debug bool) error {
	app := &application{
		logger: logger,
		debug:  debug,
	}

	if configFilename == "" {
		return fmt.Errorf("please provide a config file")
	}

	config, err := GetConfig(configFilename)
	if err != nil {
		return err
	}
	app.config = config

	app.notify = notify.New()
	var services []notify.Notifier

	if config.Notifications.Telegram.APIToken != "" {
		app.logger.Info("Notifications: using telegram")
		telegramService, err := telegram.New(config.Notifications.Telegram.APIToken)
		if err != nil {
			return fmt.Errorf("telegram setup: %w", err)
		}
		telegramService.AddReceivers(config.Notifications.Telegram.ChatIDs...)
		services = append(services, telegramService)
	}

	if config.Notifications.Discord.BotToken != "" || config.Notifications.Discord.OAuthToken != "" {
		app.logger.Info("Notifications: using discord")
		discordService := discord.New()
		if config.Notifications.Discord.BotToken != "" {
			if err := discordService.AuthenticateWithBotToken(config.Notifications.Discord.BotToken); err != nil {
				return fmt.Errorf("discord bot token setup: %w", err)
			}
		} else if config.Notifications.Discord.OAuthToken != "" {
			if err := discordService.AuthenticateWithOAuth2Token(config.Notifications.Discord.OAuthToken); err != nil {
				return fmt.Errorf("discord oauth token setup: %w", err)
			}
		} else {
			panic("logic error")
		}
		discordService.AddReceivers(config.Notifications.Discord.ChannelIDs...)
		services = append(services, discordService)
	}

	if app.config.Notifications.Email.Server != "" {
		app.logger.Info("Notifications: using email")
		mailHost := net.JoinHostPort(app.config.Notifications.Email.Server, strconv.Itoa(app.config.Notifications.Email.Port))
		mailService := mail.New(app.config.Notifications.Email.Sender, mailHost)
		if app.config.Notifications.Email.Username != "" && app.config.Notifications.Email.Password != "" {
			mailService.AuthenticateSMTP(
				"",
				app.config.Notifications.Email.Username,
				app.config.Notifications.Email.Password,
				app.config.Notifications.Email.Server,
			)
		}
		mailService.AddReceivers(app.config.Notifications.Email.Recipients...)
		services = append(services, mailService)
	}

	if config.Notifications.SendGrid.APIKey != "" {
		app.logger.Info("Notifications: using sendgrid")
		sendGridService := sendgrid.New(
			config.Notifications.SendGrid.APIKey,
			config.Notifications.SendGrid.SenderAddress,
			config.Notifications.SendGrid.SenderName,
		)
		sendGridService.AddReceivers(config.Notifications.SendGrid.Recipients...)
		services = append(services, sendGridService)
	}

	if config.Notifications.MSTeams.Webhooks != nil && len(config.Notifications.MSTeams.Webhooks) > 0 {
		app.logger.Info("Notifications: using msteams")
		msteamsService := msteams.New()
		msteamsService.AddReceivers(config.Notifications.MSTeams.Webhooks...)
		services = append(services, msteamsService)
	}

	app.notify.UseServices(services...)

	app.logger.Info("Starting server",
		slog.String("host", config.Server.Listen),
		slog.Duration("gracefultimeout", config.Server.GracefulTimeout),
		slog.Duration("timeout", config.Timeout),
		slog.Bool("debug", app.debug),
	)

	app.renderer = &TemplateRenderer{
		templates: template.Must(template.New("").Funcs(template.FuncMap{"StringsJoin": strings.Join}).ParseFS(staticFS, "templates/*")),
	}
	app.cache = cache.New(config.Cache.Timeout, config.Cache.Timeout)

	srv := &http.Server{
		Addr:    config.Server.Listen,
		Handler: app.routes(),
	}

	c := make(chan os.Signal, 1)

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			app.logger.Error("error on listenandserve", slog.String("err", err.Error()))
			// emit signal to kill server
			c <- os.Kill
		}
	}()

	app.logger.Info("Starting pprof server",
		slog.String("host", app.config.Server.PprofListen),
	)

	pprofSrv := &http.Server{
		Addr: app.config.Server.PprofListen,
	}
	go func() {
		metricsMux := http.NewServeMux()
		metricsMux.Handle("/debug/pprof/", http.DefaultServeMux)
		if err := pprofSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			app.logger.Error("error on pprof listenandserve", slog.String("err", err.Error()))
			// emit signal to kill server
			c <- os.Kill
		}
	}()

	signal.Notify(c, syscall.SIGTERM, syscall.SIGINT)
	<-c
	app.logger.Info("shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), config.Server.GracefulTimeout)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		app.logger.Error("error on srv shutdown", slog.String("err", err.Error()))
	}
	if err := pprofSrv.Shutdown(ctx); err != nil {
		app.logger.Error("error on pprofsrv shutdown", slog.String("err", err.Error()))
	}
	os.Exit(0)
	return nil
}

func (app *application) customHTTPErrorHandler(err error, c echo.Context) {
	if c.Response().Committed {
		return
	}

	code := http.StatusInternalServerError
	var echoError *echo.HTTPError
	if errors.As(err, &echoError) {
		code = echoError.Code
	}

	// send an asynchronous notification (but ignore 404 and stuff)
	if err != nil && code > 499 {
		app.logger.Error("error on request", slog.String("err", err.Error()))

		go func(e error) {
			app.logger.Debug("sending error notification", slog.String("err", e.Error()))
			if err2 := app.notify.Send(context.Background(), "ERROR", e.Error()); err2 != nil {
				app.logger.Error("error on notification send", slog.String("err", err2.Error()))
			}
		}(err)
	}

	// send error page
	errorPage := fmt.Sprintf("error_pages/HTTP%d.html", code)
	content, err2 := errorPages.ReadFile(errorPage)
	if err2 != nil {
		app.logger.Error("could not read error page", slog.String("err", err2.Error()))
		return
	}
	if err2 := c.HTMLBlob(code, content); err2 != nil {
		app.logger.Error("could not send error page", slog.String("err", err2.Error()))
		return
	}
}

func (app *application) routes() http.Handler {
	e := echo.New()
	e.HideBanner = true
	e.Debug = app.debug
	e.Renderer = app.renderer
	e.HTTPErrorHandler = app.customHTTPErrorHandler

	if app.config.Cloudflare {
		e.IPExtractor = extractIPFromCloudflareHeader()
	}

	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogStatus:        true,
		LogURI:           true,
		LogUserAgent:     true,
		LogLatency:       true,
		LogRemoteIP:      true,
		LogMethod:        true,
		LogContentLength: true,
		LogResponseSize:  true,
		LogError:         true,
		HandleError:      true, // forwards error to the global error handler, so it can decide appropriate status code
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			logLevel := slog.LevelInfo
			errString := ""
			// only set error on real errors
			if v.Error != nil && v.Status > 499 {
				errString = v.Error.Error()
				logLevel = slog.LevelError
			}
			app.logger.LogAttrs(context.Background(), logLevel, "REQUEST",
				slog.String("ip", v.RemoteIP),
				slog.String("method", v.Method),
				slog.String("uri", v.URI),
				slog.Int("status", v.Status),
				slog.String("user-agent", v.UserAgent),
				slog.Duration("request-duration", v.Latency),
				slog.String("request-length", v.ContentLength), // request content length
				slog.Int64("response-size", v.ResponseSize),
				slog.String("err", errString))

			return nil
		},
	}))
	e.Use(middleware.Secure())
	e.Use(middleware.RecoverWithConfig(middleware.RecoverConfig{
		LogErrorFunc: func(c echo.Context, err error, stack []byte) error {
			// send the error to the default error handler
			return fmt.Errorf("PANIC! %v - %s", err, string(stack))
		},
	}))

	static := echo.MustSubFS(staticFS, "assets")
	e.FileFS("/robots.txt", "robots.txt", static)
	e.StaticFS("/static", static)

	e.GET("/", func(c echo.Context) error {
		return c.Render(http.StatusOK, "index.html", nil)
	})
	e.GET("/test_panic", func(c echo.Context) error {
		// no checks in debug mode
		if app.debug {
			panic("test")
		}

		headerValue := c.Request().Header.Get(secretKeyHeaderName)
		if headerValue == "" {
			app.logger.Error("test_panic called without secret header")
		} else if headerValue == app.config.Notifications.SecretKeyHeader {
			panic("test")
		} else {
			app.logger.Error("test_panic called without valid header")
		}
		return c.Render(http.StatusOK, "index.html", nil)
	})
	e.GET("/test_notifications", func(c echo.Context) error {
		// no checks in debug mode
		if app.debug {
			return fmt.Errorf("test")
		}

		headerValue := c.Request().Header.Get(secretKeyHeaderName)
		if headerValue == "" {
			app.logger.Error("test_notification called without secret header")
		} else if headerValue == app.config.Notifications.SecretKeyHeader {
			return fmt.Errorf("test")
		} else {
			app.logger.Error("test_notification called without valid header")
		}
		return c.Render(http.StatusOK, "index.html", nil)
	})
	e.GET("/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "route")
	})
	return e
}

func extractIPFromCloudflareHeader() echo.IPExtractor {
	return func(req *http.Request) string {
		if realIP := req.Header.Get(cloudflareIPHeaderName); realIP != "" {
			return realIP
		}
		// fall back to normal ip extraction
		return echo.ExtractIPDirect()(req)
	}
}
