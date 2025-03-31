package handlers

import (
	"errors"
	"net/http"
	"runtime/debug"

	"github.com/labstack/echo/v4"
)

type VersionHandler struct{}

func NewVersionHandler() *VersionHandler {
	return &VersionHandler{}
}

func (h *VersionHandler) EchoHandler(c echo.Context) error {
	buildInfo, ok := debug.ReadBuildInfo()
	if !ok {
		return errors.New("unable to determine version information")
	}
	return c.String(http.StatusOK, buildInfo.String())
}
