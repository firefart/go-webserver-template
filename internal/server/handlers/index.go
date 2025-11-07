package handlers

import (
	"net/http"

	"github.com/firefart/go-webserver-template/internal/server/helper"
	"github.com/firefart/go-webserver-template/internal/server/templates"
)

type IndexHandler struct {
	debug bool
}

func NewIndexHandler(debug bool) *IndexHandler {
	return &IndexHandler{
		debug: debug,
	}
}

func (h *IndexHandler) Handler(w http.ResponseWriter, r *http.Request) error {
	component := templates.Homepage()

	w.WriteHeader(http.StatusOK)

	if helper.IsHTMX(r) {
		// only render the single component if it's a htmx request
		if err := component.Render(r.Context(), w); err != nil {
			return err
		}
		return nil
	}

	layout := templates.Layout(component, "Template", h.debug)
	if err := layout.Render(r.Context(), w); err != nil {
		return err
	}
	return nil
}
