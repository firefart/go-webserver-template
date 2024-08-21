package server

import (
	"log/slog"

	"github.com/firefart/go-webserver-template/internal/config"
	"github.com/firefart/go-webserver-template/internal/database"
	"github.com/nikoksr/notify"
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
