package database

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"strings"

	"github.com/firefart/go-webserver-template/internal/config"
	"github.com/firefart/go-webserver-template/internal/database/sqlc"
	"github.com/pressly/goose/v3"

	// use the sqlite implementation
	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

type Database struct {
	reader    *sqlc.Queries
	writer    *sqlc.Queries
	readerRAW *sql.DB
	writerRAW *sql.DB
}

func New(ctx context.Context, configuration config.Configuration, logger *slog.Logger) (*Database, error) {
	if strings.ToLower(configuration.Database.Filename) == ":memory:" {
		// not possible because of the two db instances, with in memory they
		// would be separate instances
		return nil, fmt.Errorf("in memory databases are not supported")
	}

	reader, err := newDatabase(ctx, configuration, logger, true)
	if err != nil {
		return nil, fmt.Errorf("could not create reader: %w", err)
	}
	reader.SetMaxOpenConns(100)
	// no migrations on the second connection
	writer, err := newDatabase(ctx, configuration, logger, false)
	if err != nil {
		return nil, fmt.Errorf("could not create writer: %w", err)
	}
	// only one writer connection as there can only be one
	writer.SetMaxOpenConns(1)
	writer.SetMaxIdleConns(1)

	return &Database{
		reader:    sqlc.New(reader),
		writer:    sqlc.New(writer),
		readerRAW: reader,
		writerRAW: writer,
	}, nil
}

func newDatabase(ctx context.Context, configuration config.Configuration, logger *slog.Logger, skipMigrations bool) (*sql.DB, error) {
	db, err := sql.Open("sqlite", fmt.Sprintf("%s?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)", configuration.Database.Filename))
	if err != nil {
		return nil, fmt.Errorf("could not open database %s: %w", configuration.Database.Filename, err)
	}

	// we have a reader and a writer so no need to apply all migrations twice
	if !skipMigrations {
		migrationFS, err := fs.Sub(embedMigrations, "migrations")
		if err != nil {
			return nil, fmt.Errorf("could not sub migration fs: %w", err)
		}

		prov, err := goose.NewProvider("sqlite3", db, migrationFS)
		if err != nil {
			return nil, fmt.Errorf("could not create goose provider: %w", err)
		}

		result, err := prov.Up(ctx)
		if err != nil {
			return nil, fmt.Errorf("could not apply migrations: %w", err)
		}

		for _, r := range result {
			if r.Error != nil {
				return nil, fmt.Errorf("could not apply migration %s: %w", r.Source.Path, r.Error)
			}
		}

		if len(result) > 0 {
			logger.Info(fmt.Sprintf("applied %d database migrations", len(result)))
		}

		version, err := prov.GetDBVersion(ctx)
		if err != nil {
			return nil, fmt.Errorf("could not get current database version: %w", err)
		}
		logger.Info("database setup completed", slog.Int64("version", version))
	}

	// shrink and defrag the database (must be run before the checkpoint)
	if _, err := db.Exec("VACUUM;"); err != nil {
		return nil, fmt.Errorf("could not vacuum: %w", err)
	}

	// truncate the wal file
	if _, err := db.Exec("PRAGMA wal_checkpoint(TRUNCATE);"); err != nil {
		return nil, fmt.Errorf("could not truncate wal: %w", err)
	}

	// set synchronous mode to normal as it's recommended for WAL
	if _, err := db.Exec("PRAGMA synchronous(NORMAL);"); err != nil {
		return nil, fmt.Errorf("could not set synchronous: %w", err)
	}

	// set the busy timeout (ms) - how long a command waits to be executed when the db is locked / busy
	if _, err := db.Exec("PRAGMA busy_timeout(5000);"); err != nil {
		return nil, fmt.Errorf("could not set synchronous: %w", err)
	}

	return db, nil
}

func (db *Database) Close() error {
	err1 := db.writerRAW.Close()
	err2 := db.readerRAW.Close()
	return errors.Join(err1, err2)
}
