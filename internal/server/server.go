package server

import (
	"context"
	"embed"
	"io"
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
	db     database.Interface
	notify *notify.Notify
	debug  bool
}

//go:embed assets
var fsAssets embed.FS

func NewServer(ctx context.Context, opts ...OptionsServerFunc) http.Handler {
	// func NewServer(ctx context.Context, logger *slog.Logger, config config.Configuration, db database.DatabaseInterface, notify *notify.Notify, debug bool) http.Handler {
	s := server{
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		debug:  false,
	}

	for _, o := range opts {
		o(&s)
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
