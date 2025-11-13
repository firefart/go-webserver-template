package server

import (
	"log/slog"

	"github.com/firefart/go-webserver-template/internal/cacher"
	"github.com/firefart/go-webserver-template/internal/config"
	"github.com/firefart/go-webserver-template/internal/database"
	"github.com/firefart/go-webserver-template/internal/http"
	"github.com/firefart/go-webserver-template/internal/mail"
	"github.com/firefart/go-webserver-template/internal/metrics"

	"github.com/nikoksr/notify"
)

type OptionsServerFunc func(c *server) error

func WithLogger(logger *slog.Logger) OptionsServerFunc {
	return func(c *server) error { c.logger = logger; return nil }
}

func WithConfig(config config.Configuration) OptionsServerFunc {
	return func(c *server) error { c.config = config; return nil }
}

func WithNotify(n *notify.Notify) OptionsServerFunc {
	return func(c *server) error { c.notify = n; return nil }
}

func WithDB(db database.Interface) OptionsServerFunc {
	return func(c *server) error { c.db = db; return nil }
}

func WithDebug(d bool) OptionsServerFunc {
	return func(c *server) error { c.debug = d; return nil }
}

func WithMetrics(m *metrics.Metrics) OptionsServerFunc {
	return func(c *server) error { c.metrics = m; return nil }
}

func WithAccessLog() OptionsServerFunc {
	return func(c *server) error { c.accessLog = true; return nil }
}

func WithCache(cache *cacher.Cache[string]) OptionsServerFunc {
	return func(c *server) error { c.cache = cache; return nil }
}

func WithHTTPClient(client *http.Client) OptionsServerFunc {
	return func(c *server) error { c.httpClient = client; return nil }
}

func WithMailer(mailer mail.Interface) OptionsServerFunc {
	return func(c *server) error { c.mailer = mailer; return nil }
}
