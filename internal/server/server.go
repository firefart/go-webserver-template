package server

import (
	"context"
	"embed"
	"io"
	"log/slog"
	"net/http"

	"github.com/firefart/go-webserver-template/internal/config"
	"github.com/firefart/go-webserver-template/internal/database"
	custommiddleware "github.com/firefart/go-webserver-template/internal/server/middleware"
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
	s := server{
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		debug:  false,
	}

	for _, o := range opts {
		o(&s)
	}

	e := echo.New()
	e.HideBanner = true
	e.HTTPErrorHandler = s.customHTTPErrorHandler

	if s.config.Cloudflare {
		e.IPExtractor = extractIPFromCloudflareHeader()
	}

	e.Use(custommiddleware.RequestLogger(ctx, s.logger))
	e.Use(middleware.Secure())
	e.Use(custommiddleware.Recover())

	// add all the routes
	s.addRoutes(e)
	return e
}
