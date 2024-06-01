package server

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"

	"github.com/labstack/echo/v4"
)

var cloudflareIPHeaderName = http.CanonicalHeaderKey("CF-Connecting-IP")

func (s *Server) customHTTPErrorHandler(err error, c echo.Context) {
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
		s.logger.Error("error on request", slog.String("err", err.Error()))

		go func(e error) {
			s.logger.Debug("sending error notification", slog.String("err", e.Error()))
			if err2 := s.notify.Send(context.Background(), "ERROR", e.Error()); err2 != nil {
				s.logger.Error("error on notification send", slog.String("err", err2.Error()))
			}
		}(err)
	}

	// send error page
	errorPage := fmt.Sprintf("assets/error_pages/HTTP%d.html", code)
	if _, err := fs.Stat(fsAssets, errorPage); err == nil {
		// file exists, no further processing
	} else if errors.Is(err, os.ErrNotExist) {
		errorPage = "error_pages/HTTP500.html"
	} else {
		s.logger.Error("could not check if file exists", slog.String("err", err.Error()))
		errorPage = "error_pages/HTTP500.html"
	}

	content, err2 := fsAssets.ReadFile(errorPage)
	if err2 != nil {
		s.logger.Error("could not read error page", slog.String("err", err2.Error()))
		return
	}
	if err2 := c.HTMLBlob(code, content); err2 != nil {
		s.logger.Error("could not send error page", slog.String("err", err2.Error()))
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
