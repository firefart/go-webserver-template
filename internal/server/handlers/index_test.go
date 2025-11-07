package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/firefart/go-webserver-template/internal/server/handlers"
	"github.com/stretchr/testify/require"
)

func TestIndex(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	require.NoError(t, handlers.NewIndexHandler(true).Handler(rec, req))
	require.Equal(t, http.StatusOK, rec.Code)
	require.Greater(t, len(rec.Body.String()), 10)
}
