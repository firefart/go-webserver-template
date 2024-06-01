package handlers

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
)

type NotificationHandler struct {
	debug                bool
	logger               *slog.Logger
	secretKeyHeaderName  string
	secretKeyHeaderValue string
}

func NewNotificationHandler(logger *slog.Logger, debug bool, secretKeyHeaderName, secretKeyHeaderValue string) *NotificationHandler {
	return &NotificationHandler{
		debug:                debug,
		logger:               logger,
		secretKeyHeaderName:  secretKeyHeaderName,
		secretKeyHeaderValue: secretKeyHeaderValue,
	}
}

func (h *NotificationHandler) EchoHandler(c echo.Context) error {
	// no checks in debug mode
	if h.debug {
		return fmt.Errorf("test")
	}

	headerValue := c.Request().Header.Get(h.secretKeyHeaderName)
	if headerValue == "" {
		h.logger.Error("test_notification called without secret header")
	} else if headerValue == h.secretKeyHeaderValue {
		return fmt.Errorf("test")
	} else {
		h.logger.Error("test_notification called without valid header")
	}
	return c.NoContent(http.StatusOK)
}
