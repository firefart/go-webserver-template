package middleware

import (
	"io"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type SecretKeyHeaderConfig struct {
	// Skipper defines a function to skip middleware.
	Skipper middleware.Skipper

	// the secret key header name we should check
	SecretKeyHeaderName  string
	SecretKeyHeaderValue string

	Logger *slog.Logger
}

func SecretKeyHeader(config SecretKeyHeaderConfig) echo.MiddlewareFunc {
	// Defaults
	if config.SecretKeyHeaderName == "" {
		panic("secret key header middleware requires a header name")
	}
	if config.SecretKeyHeaderValue == "" {
		panic("secret key header middleware requires a header value")
	}
	if config.Skipper == nil {
		config.Skipper = middleware.DefaultSkipper
	}
	if config.Logger == nil {
		config.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if config.Skipper(c) {
				return next(c)
			}

			headerVal := c.Request().Header.Get(config.SecretKeyHeaderName)
			// no header set
			if headerVal == "" {
				config.Logger.Error("url called without secret header", slog.String("url", c.Request().URL.String()))
				return c.NoContent(http.StatusOK)
			}

			if headerVal == config.SecretKeyHeaderValue {
				return next(c)
			}

			config.Logger.Error("url called with wrong secret header", slog.String("header", headerVal))
			return c.NoContent(http.StatusOK)
		}
	}
}
