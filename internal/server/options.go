package server

import (
	"log/slog"

	"github.com/firefart/go-webserver-template/internal/cacher"
	"github.com/firefart/go-webserver-template/internal/config"
	"github.com/firefart/go-webserver-template/internal/database"
	"github.com/firefart/go-webserver-template/internal/metrics"

	"github.com/nikoksr/notify"
	"github.com/prometheus/client_golang/prometheus"
)

type OptionsServerFunc func(c *server)

func WithLogger(logger *slog.Logger) OptionsServerFunc {
	return func(c *server) { c.logger = logger }
}

func WithConfig(config config.Configuration) OptionsServerFunc {
	return func(c *server) { c.config = config }
}

func WithNotify(n *notify.Notify) OptionsServerFunc {
	return func(c *server) { c.notify = n }
}

func WithDB(db database.Interface) OptionsServerFunc {
	return func(c *server) { c.db = db }
}

func WithDebug(d bool) OptionsServerFunc {
	return func(c *server) { c.debug = d }
}

func WithMetrics(m *metrics.Metrics, reg prometheus.Registerer) OptionsServerFunc {
	return func(c *server) {
		c.metrics = m
		c.promRegistry = reg
	}
}

func WithCache(cache *cacher.Cache[string]) OptionsServerFunc {
	return func(c *server) {
		c.cache = cache
	}
}
