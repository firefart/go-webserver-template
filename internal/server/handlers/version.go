package handlers

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/labstack/echo/v4"
)

type VersionHandler struct {
	debug                bool
	logger               *slog.Logger
	secretKeyHeaderName  string
	secretKeyHeaderValue string
}

func NewVersionHandler(logger *slog.Logger, debug bool, secretKeyHeaderName, secretKeyHeaderValue string) *VersionHandler {
	return &VersionHandler{
		debug:                debug,
		logger:               logger,
		secretKeyHeaderName:  secretKeyHeaderName,
		secretKeyHeaderValue: secretKeyHeaderValue,
	}
}

func (h *VersionHandler) EchoHandler(c echo.Context) error {
	// no checks in debug mode
	if h.debug {
		buildInfo, ok := debug.ReadBuildInfo()
		if !ok {
			return fmt.Errorf("Unable to determine version information")
		}
		return c.String(http.StatusOK, buildInfo.String())
	}

	headerValue := c.Request().Header.Get(h.secretKeyHeaderName)
	if headerValue == "" {
		h.logger.Error("version info called without secret header")
	} else if headerValue == h.secretKeyHeaderValue {
		buildInfo, ok := debug.ReadBuildInfo()
		if !ok {
			return fmt.Errorf("Unable to determine version information")
		}
		return c.String(http.StatusOK, buildInfo.String())
	} else {
		h.logger.Error("version info called without valid header")
	}
	return c.NoContent(http.StatusOK)
}
