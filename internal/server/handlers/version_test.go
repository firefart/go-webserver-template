package handlers_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/firefart/go-webserver-template/internal/config"
	"github.com/firefart/go-webserver-template/internal/server"
	"github.com/firefart/go-webserver-template/internal/server/handlers"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

func TestVersion(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	configuration := config.Configuration{
		SecretKeyHeaderName:  "X-Secret-Key",
		SecretKeyHeaderValue: "SECRET",
	}

	e := server.NewServer(ctx, server.WithConfig(configuration))
	x, ok := e.(*echo.Echo)
	require.True(t, ok)

	// test debug mode
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	cont := x.NewContext(req, rec)
	require.Nil(t, handlers.NewVersionHandler(logger, true, "", "").EchoHandler(cont))
	require.Equal(t, http.StatusOK, rec.Code)
	require.Greater(t, len(rec.Body.String()), 10)

	// test normal mode without debug and valid header
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/version", nil)
	req.Header.Set(configuration.SecretKeyHeaderName, configuration.SecretKeyHeaderValue)
	cont = x.NewContext(req, rec)
	require.Nil(t, handlers.NewVersionHandler(logger, false, configuration.SecretKeyHeaderName, configuration.SecretKeyHeaderValue).EchoHandler(cont))
	require.Equal(t, http.StatusOK, rec.Code)
	require.Greater(t, len(rec.Body.String()), 10)

	// test normal mode without debug and invalid header value
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/version", nil)
	req.Header.Set(configuration.SecretKeyHeaderName, "INVALID")
	cont = x.NewContext(req, rec)
	require.Nil(t, handlers.NewVersionHandler(logger, false, configuration.SecretKeyHeaderName, configuration.SecretKeyHeaderValue).EchoHandler(cont))
	require.Equal(t, http.StatusOK, rec.Code)
	require.Len(t, rec.Body.String(), 0)

	// test normal mode without debug and no header
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/version", nil)
	cont = x.NewContext(req, rec)
	require.Nil(t, handlers.NewVersionHandler(logger, false, configuration.SecretKeyHeaderName, configuration.SecretKeyHeaderValue).EchoHandler(cont))
	require.Equal(t, http.StatusOK, rec.Code)
	require.Len(t, rec.Body.String(), 0)
}
