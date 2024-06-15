package handlers_test

import (
	"context"
	"io"
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
	ctx := context.Background()
	db := database.NewMockDB()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	c := config.Configuration{}

	e := server.NewServer(ctx, logger, c, db, nil, false)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	x, ok := e.(*echo.Echo)
	require.True(t, ok)
	cont := x.NewContext(req, rec)
	require.Nil(t, handlers.NewIndexHandler(true).EchoHandler(cont))
	require.Equal(t, http.StatusOK, rec.Code)
	require.Greater(t, len(rec.Body.String()), 10)
}

func TestIndex(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	file, err := os.CreateTemp("", "*.sqlite")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(file.Name())

	configuration := config.Configuration{
		Database: config.ConfigDatabase{
			Filename: file.Name(),
		},
	}

	db, err := database.New(ctx, configuration, logger)
	require.Nil(t, err)

	e := server.NewServer(ctx, logger, configuration, db, nil, false)
	x, ok := e.(*echo.Echo)
	require.True(t, ok)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	cont := x.NewContext(req, rec)
	require.Nil(t, handlers.NewIndexHandler(true).EchoHandler(cont))
	require.Equal(t, http.StatusOK, rec.Code)
	require.Greater(t, len(rec.Body.String()), 10)
}
