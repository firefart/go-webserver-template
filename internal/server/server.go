package server

import (
	"context"
	"embed"
	"log/slog"
	"net/http"

	"github.com/firefart/go-webserver-template/internal/config"
	"github.com/firefart/go-webserver-template/internal/database"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/nikoksr/notify"
)

type server struct {
	logger *slog.Logger
	config config.Configuration
	db     database.DatabaseInterface
	notify *notify.Notify
	debug  bool
}

//go:embed assets
var fsAssets embed.FS

func NewServer(ctx context.Context, logger *slog.Logger, config config.Configuration, db database.DatabaseInterface, notify *notify.Notify, debug bool) http.Handler {
	s := server{
		logger: logger,
		config: config,
		notify: notify,
		debug:  debug,
		db:     db,
	}

	e := echo.New()
	e.HideBanner = true
	e.Debug = s.debug
	e.HTTPErrorHandler = s.customHTTPErrorHandler

	if s.config.Cloudflare {
		e.IPExtractor = extractIPFromCloudflareHeader()
	}

	e.Use(s.middlewareRequestLogger(ctx))
	e.Use(middleware.Secure())
	e.Use(s.middlewareRecover())

	// add all the routes
	s.addRoutes(e)
	return e
}
