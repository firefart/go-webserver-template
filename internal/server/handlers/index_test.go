package handlers_test

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/firefart/go-webserver-template/internal/config"
	"github.com/firefart/go-webserver-template/internal/database"
	"github.com/firefart/go-webserver-template/internal/server"
	"github.com/firefart/go-webserver-template/internal/server/handlers"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

func TestIndexMock(t *testing.T) {
	db := database.NewMockDB()
	configuration := config.Configuration{
		Server: config.Server{
			SecretKeyHeaderName:  "X-Secret-Key",
			SecretKeyHeaderValue: "SECRET",
		},
	}
	e := server.NewServer(t.Context(), server.WithDB(db), server.WithConfig(configuration))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	x, ok := e.(*echo.Echo)
	require.True(t, ok)
	cont := x.NewContext(req, rec)
	require.NoError(t, handlers.NewIndexHandler(true).EchoHandler(cont))
	require.Equal(t, http.StatusOK, rec.Code)
	require.Greater(t, len(rec.Body.String()), 10)
}

func TestIndex(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)

	file, err := os.CreateTemp(t.TempDir(), "*.sqlite")
	require.NoError(t, err)
	defer func(name string) {
		err := os.Remove(name)
		require.NoError(t, err)
	}(file.Name())

	configuration := config.Configuration{
		Database: config.Database{
			Filename: file.Name(),
		},
		Server: config.Server{
			SecretKeyHeaderName:  "X-Secret-Key",
			SecretKeyHeaderValue: "SECRET",
		},
	}

	db, err := database.New(t.Context(), configuration, logger, false)
	require.NoError(t, err)

	e := server.NewServer(t.Context(), server.WithConfig(configuration), server.WithDB(db))
	x, ok := e.(*echo.Echo)
	require.True(t, ok)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	cont := x.NewContext(req, rec)
	require.NoError(t, handlers.NewIndexHandler(true).EchoHandler(cont))
	require.Equal(t, http.StatusOK, rec.Code)
	require.Greater(t, len(rec.Body.String()), 10)
}
