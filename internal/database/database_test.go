package database

import (
	"database/sql"
	"errors"
	"io/fs"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/firefart/go-webserver-template/internal/config"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "modernc.org/sqlite"
)

func TestNew(t *testing.T) {
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
	db, err := New(t.Context(), configuration, slog.New(slog.DiscardHandler), false)
	require.NoError(t, err)
	err = db.Close(1 * time.Second)
	require.NoError(t, err)
}

func TestMigrations(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("sqlite", ":memory:?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)")
	require.NoError(t, err, "could not open database")

	migrationFS, err := fs.Sub(embedMigrations, "migrations")
	require.NoError(t, err, "could not sub migration fs")

	prov, err := goose.NewProvider("sqlite3", db, migrationFS)
	require.NoError(t, err, "could not create goose provider")

	result, err := prov.Up(t.Context())
	if err != nil {
		var partialError *goose.PartialError
		switch {
		case errors.As(err, &partialError):
			require.NoError(t, partialError.Err, "could not apply migrations")
		default:
			require.NoError(t, err, "could not apply migrations")
		}
		return
	}

	for _, r := range result {
		if r.Error != nil {
			require.NoErrorf(t, r.Error, "could not apply migration %s", r.Source.Path)
		}
	}

	require.NotEmpty(t, result)

	result, err = prov.DownTo(t.Context(), 0)
	if err != nil {
		var partialError *goose.PartialError
		switch {
		case errors.As(err, &partialError):
			require.NoError(t, partialError.Err, "could not roll back migrations")
		default:
			require.NoError(t, err, "could not roll back migrations")
		}
		return
	}

	for _, r := range result {
		require.NoErrorf(t, r.Error, "could not roll back migration %s", r.Source.Path)
	}

	// check for leftover indexes
	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type = 'index'")
	require.NoError(t, err)
	defer func() {
		err := rows.Close()
		require.NoError(t, err)
	}()

	var indexNames []string
	for rows.Next() {
		var name string
		err = rows.Scan(&name)
		require.NoError(t, err)
		indexNames = append(indexNames, name)
	}
	require.NoError(t, rows.Err())

	assert.Empty(t, indexNames, "found undeleted indexes")

	// check for leftover tables
	rows, err = db.Query("SELECT name FROM sqlite_master WHERE type = 'table' and name != 'goose_db_version' and name != 'sqlite_sequence'")
	require.NoError(t, err)
	defer func() {
		err := rows.Close()
		require.NoError(t, err)
	}()

	var tableNames []string
	for rows.Next() {
		var name string
		err = rows.Scan(&name)
		require.NoError(t, err)
		tableNames = append(tableNames, name)
	}
	require.NoError(t, rows.Err())

	assert.Empty(t, tableNames, "found undeleted tables")
}
