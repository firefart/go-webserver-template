package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

func TestSecretKeyHeader(t *testing.T) {
	e := echo.New()
	handler := func(c echo.Context) error {
		return c.String(http.StatusOK, "secret content")
	}
	mwConfig := SecretKeyHeaderConfig{
		SecretKeyHeaderName:  "X-Secret-Key",
		SecretKeyHeaderValue: "secret",
	}
	mw := SecretKeyHeader(mwConfig)

	// valid header
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(mwConfig.SecretKeyHeaderName, mwConfig.SecretKeyHeaderValue)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	err := mw(handler)(c)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "secret content", rec.Body.String())

	// no header
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	err = mw(handler)(c)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Empty(t, rec.Body.String())

	// wrong header value
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(mwConfig.SecretKeyHeaderName, "wrong value")
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	err = mw(handler)(c)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Empty(t, rec.Body.String())

	// debug should skip checks
	mw = SecretKeyHeader(SecretKeyHeaderConfig{
		Skipper: func(_ echo.Context) bool {
			return true // simulate debug set to true
		},
		SecretKeyHeaderName:  "X-Secret-Key",
		SecretKeyHeaderValue: "secret",
	})
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(mwConfig.SecretKeyHeaderName, "wrong value")
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	err = mw(handler)(c)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "secret content", rec.Body.String())

	// panic if no header name set
	// debug should skip checks
	require.Panics(t, func() {
		SecretKeyHeader(SecretKeyHeaderConfig{
			SecretKeyHeaderValue: "secret",
		})
	})

	// panic if no header value set
	// debug should skip checks
	require.Panics(t, func() {
		SecretKeyHeader(SecretKeyHeaderConfig{
			SecretKeyHeaderName: "X-Secret-Key",
		})
	})
}
