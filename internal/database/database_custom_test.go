package database_test

import (
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/firefart/go-webserver-template/internal/config"
	"github.com/firefart/go-webserver-template/internal/database"
	"github.com/stretchr/testify/require"
)

func TestGetAllDummy(t *testing.T) {
	t.Parallel()

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
	}
	db, err := database.New(t.Context(), configuration, slog.New(slog.DiscardHandler), false)
	require.NoError(t, err)
	defer func(db *database.Database, timeout time.Duration) {
		err := db.Close(timeout)
		require.NoError(t, err)
	}(db, 1*time.Second)

	id, err := db.InsertDummy(t.Context(), "Test")
	require.NoError(t, err)
	require.Positive(t, id)

	ids, err := db.GetAllDummy(t.Context())
	require.NoError(t, err)
	require.Len(t, ids, 1)
	require.Equal(t, id, ids[0])
}
