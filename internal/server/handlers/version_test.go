package handlers_test

import (
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
	configuration := config.Configuration{
		Server: config.Server{
			SecretKeyHeaderName:  "X-Secret-Key",
			SecretKeyHeaderValue: "SECRET",
		},
	}

	e := server.NewServer(t.Context(), server.WithConfig(configuration))
	x, ok := e.(*echo.Echo)
	require.True(t, ok)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	cont := x.NewContext(req, rec)
	require.NoError(t, handlers.NewVersionHandler().EchoHandler(cont))
	require.Equal(t, http.StatusOK, rec.Code)
	require.Greater(t, len(rec.Body.String()), 10)
}
