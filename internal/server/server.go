package server

import (
	"context"
	"embed"
	"log/slog"
	nethttp "net/http"

	"github.com/firefart/go-webserver-template/internal/cacher"
	"github.com/firefart/go-webserver-template/internal/config"
	"github.com/firefart/go-webserver-template/internal/database"
	"github.com/firefart/go-webserver-template/internal/http"
	"github.com/firefart/go-webserver-template/internal/mail"
	"github.com/firefart/go-webserver-template/internal/metrics"
	custommiddleware "github.com/firefart/go-webserver-template/internal/server/middleware"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/nikoksr/notify"
	"github.com/prometheus/client_golang/prometheus"
)

type server struct {
	logger       *slog.Logger
	config       config.Configuration
	db           database.Interface
	notify       *notify.Notify
	metrics      *metrics.Metrics
	promRegistry prometheus.Registerer
	cache        *cacher.Cache[string]
	mailer       mail.Interface
	httpClient   *http.Client
	debug        bool
}

//go:embed assets
var fsAssets embed.FS

func NewServer(ctx context.Context, opts ...OptionsServerFunc) nethttp.Handler {
	s := server{
		logger: slog.New(slog.DiscardHandler),
		debug:  false,
	}

	for _, o := range opts {
		o(&s)
	}

	e := echo.New()
	e.HideBanner = true
	e.HTTPErrorHandler = s.customHTTPErrorHandler

	if s.config.Server.Cloudflare {
		e.IPExtractor = extractIPFromCloudflareHeader()
	}

	e.Use(custommiddleware.RequestLogger(ctx, s.logger))
	e.Use(middleware.Secure())
	e.Use(custommiddleware.Recover())

	// add all the routes
	s.addRoutes(e)
	return e
}
