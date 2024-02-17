package main

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

//go:embed templates
//go:embed assets
var staticFS embed.FS

//go:embed error_pages
var errorPages embed.FS

// TemplateRenderer is a custom html/template renderer for Echo framework
type TemplateRenderer struct {
	templates *template.Template
}

// Render renders a template document
func (t *TemplateRenderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

func (app *application) newServer() http.Handler {
	e := echo.New()
	e.HideBanner = true
	e.Debug = app.debug
	e.Renderer = &TemplateRenderer{
		templates: template.Must(template.New("").Funcs(getTemplateFuncMap()).ParseFS(staticFS, "templates/*")),
	}
	e.HTTPErrorHandler = app.customHTTPErrorHandler

	if app.config.Cloudflare {
		e.IPExtractor = extractIPFromCloudflareHeader()
	}

	e.Use(app.middlewareRequestLogger())
	e.Use(middleware.Secure())
	e.Use(app.middlewareRecover())

	// add all the routes
	app.addRoutes(e)
	return e
}

func (app *application) customHTTPErrorHandler(err error, c echo.Context) {
	if c.Response().Committed {
		return
	}

	code := http.StatusInternalServerError
	var echoError *echo.HTTPError
	if errors.As(err, &echoError) {
		code = echoError.Code
	}

	// send an asynchronous notification (but ignore 404 and stuff)
	if err != nil && code > 499 {
		app.logger.Error("error on request", slog.String("err", err.Error()))

		go func(e error) {
			app.logger.Debug("sending error notification", slog.String("err", e.Error()))
			if err2 := app.notify.Send(context.Background(), "ERROR", e.Error()); err2 != nil {
				app.logger.Error("error on notification send", slog.String("err", err2.Error()))
			}
		}(err)
	}

	// send error page
	errorPage := fmt.Sprintf("error_pages/HTTP%d.html", code)
	content, err2 := errorPages.ReadFile(errorPage)
	if err2 != nil {
		app.logger.Error("could not read error page", slog.String("err", err2.Error()))
		return
	}
	if err2 := c.HTMLBlob(code, content); err2 != nil {
		app.logger.Error("could not send error page", slog.String("err", err2.Error()))
		return
	}
}

func extractIPFromCloudflareHeader() echo.IPExtractor {
	return func(req *http.Request) string {
		if realIP := req.Header.Get(cloudflareIPHeaderName); realIP != "" {
			return realIP
		}
		// fall back to normal ip extraction
		return echo.ExtractIPDirect()(req)
	}
}
