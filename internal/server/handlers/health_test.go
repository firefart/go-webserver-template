package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/firefart/go-webserver-template/internal/server/handlers"
	"github.com/stretchr/testify/require"
)

func TestHealth(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	require.NoError(t, handlers.NewHealthHandler().Handler(rec, req))
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "OK", rec.Body.String())
}
