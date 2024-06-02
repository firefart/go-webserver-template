package handlers

import (
	"net/http"

	"github.com/firefart/go-webserver-template/internal/server/templates"
	"github.com/labstack/echo/v4"
)

type IndexHandler struct {
	debug bool
}

func NewIndexHandler(debug bool) *IndexHandler {
	return &IndexHandler{
		debug: debug,
	}
}

func (h *IndexHandler) EchoHandler(c echo.Context) error {
	component := templates.Homepage()

	if isHTMX(c) {
		// only render the single component if it's a htmx request
		return Render(c, http.StatusOK, component)
	}

	return Render(c, http.StatusOK, templates.Layout(component, "Template", h.debug))
}
