package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
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

	_ "go.uber.org/automaxprocs"
)

//go:embed templates
//go:embed assets
var staticFS embed.FS

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
	w := os.Stdout
	var level = new(slog.LevelVar)
	level.Set(slog.LevelInfo)
	logger := slog.New(tint.NewHandler(w, &tint.Options{
		Level:   level,
		NoColor: !isatty.IsTerminal(w.Fd()),
	}))
	if err := run(logger, level); err != nil {
		trace := string(debug.Stack())
		logger.Error(err.Error(), "trace", trace)
		os.Exit(1)
	}
}

func run(logger *slog.Logger, level *slog.LevelVar) error {
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
		level.Set(slog.LevelDebug)
		app.debug = true
	}

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
		slog.Int("port", config.Server.Port),
		slog.Duration("gracefultimeout", config.Server.GracefulTimeout),
		slog.Duration("timeout", config.Timeout),
		slog.Bool("debug", app.debug),
	)

	app.renderer = &TemplateRenderer{
		templates: template.Must(template.New("").Funcs(template.FuncMap{"StringsJoin": strings.Join}).ParseFS(staticFS, "templates/*")),
	}
	app.cache = cache.New(config.Cache.Timeout, config.Cache.Timeout)

	srv := &http.Server{
		Addr:    net.JoinHostPort(config.Server.Listen, strconv.Itoa(config.Server.Port)),
		Handler: app.routes(),
	}

	go func() {
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			app.logger.Error(err.Error(), "trace", string(debug.Stack()))
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGTERM, syscall.SIGINT)
	<-c
	ctx, cancel := context.WithTimeout(context.Background(), config.Server.GracefulTimeout)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		app.logger.Error(err.Error(), "trace", string(debug.Stack()))
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
		e.IPExtractor = extractIPFromCloudflareHeader()
	}

	e.Use(middleware.Logger())
	e.Use(middleware.Secure())
	e.Use(middleware.Recover())

	static := echo.MustSubFS(staticFS, "assets")
	e.FileFS("/robots.txt", "robots.txt", static)
	e.StaticFS("/static", static)

	e.GET("/", func(c echo.Context) error {
		return c.Render(http.StatusOK, "index.html", nil)
	})
	e.GET("/test_notifications", func(c echo.Context) error {
		headerValue := c.Request().Header.Get(secretKeyHeaderName)
		if headerValue == "" {
			app.logger.Error("test_notification called without secret header")
		} else if headerValue == app.config.Notifications.SecretKeyHeader {
			app.logEror(fmt.Errorf("test"))
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

func (app *application) logEror(err error) {
	app.logger.Error(err.Error(), "trace", string(debug.Stack()))
	if err2 := app.notify.Send(context.Background(), "[ERROR]", err.Error()); err != nil {
		app.logger.Error(err2.Error(), "trace", string(debug.Stack()))
	}
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
