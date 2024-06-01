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
	return Render(c, http.StatusOK, templates.Layout(component, "Template", h.debug))
}
