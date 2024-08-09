package database

import (
	"context"
	"database/sql"
	"errors"
	"io/fs"
	"testing"

	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"

	_ "modernc.org/sqlite"
)

func TestMigrations(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)")
	require.Nil(t, err, "could not open database")

	migrationFS, err := fs.Sub(embedMigrations, "migrations")
	require.Nil(t, err, "could not sub migration fs")

	prov, err := goose.NewProvider("sqlite3", db, migrationFS)
	require.Nil(t, err, "could not create goose provider")

	ctx := context.Background()

	result, err := prov.Up(ctx)
	require.Nil(t, err, "could not apply migrations")

	for _, r := range result {
		if r.Error != nil {
			require.Nilf(t, r.Error, "could not apply migration %s", r.Source.Path)
		}
	}

	require.Positive(t, len(result))

	result, err = prov.DownTo(ctx, 0)
	if err != nil {
		var partialError *goose.PartialError
		switch {
		case errors.As(err, &partialError):
			require.Nil(t, partialError.Err, "could not roll back migrations")
		default:
			require.Nil(t, err, "could not roll back migrations")
		}
		return
	}

	for _, r := range result {
		if r.Error != nil {
			require.Nilf(t, r.Error, "could not roll back migration %s", r.Source.Path)
		}
	}
}
