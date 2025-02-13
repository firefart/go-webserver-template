package database_test

import (
	"log"
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

	file, err := os.CreateTemp("", "*.sqlite")
	if err != nil {
		log.Fatal(err)
	}
	defer func(name string) {
		err := os.Remove(name)
		if err != nil {
			require.Nil(t, err)
		}
	}(file.Name())

	configuration := config.Configuration{
		Database: config.Database{
			Filename: file.Name(),
		},
	}
	db, err := database.New(t.Context(), configuration, slog.New(slog.DiscardHandler), false)
	require.Nil(t, err)
	defer func(db *database.Database, timeout time.Duration) {
		err := db.Close(timeout)
		if err != nil {
			require.Nil(t, err)
		}
	}(db, 1*time.Second)

	id, err := db.InsertDummy(t.Context(), "Test")
	require.Nil(t, err)
	require.Positive(t, id)

	ids, err := db.GetAllDummy(t.Context())
	require.Nil(t, err)
	require.Len(t, ids, 1)
	require.Equal(t, id, ids[0])
}
