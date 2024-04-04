package database

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/firefart/go-webserver-template/internal/config"

	_ "modernc.org/sqlite"
)

const create string = `
	CREATE TABLE IF NOT EXISTS DUMMY (
		ID INTEGER NOT NULL PRIMARY KEY,
		NAME TEXT NOT NULL,
	) STRICT;
	CREATE UNIQUE INDEX IF NOT EXISTS IDX_NAME
	ON DUMMY(NAME);
`

var ErrNotFound = errors.New("record not found in database")

type Database struct {
	reader *sql.DB
	writer *sql.DB
}

func New(configuration config.Configuration) (*Database, error) {
	if strings.ToLower(configuration.Database.Filename) == ":memory:" {
		// not possible because of the two db instances, with in memory they
		// would be separate instances
		return nil, fmt.Errorf("in memory databases are not supported")
	}

	reader, err := newDatabase(configuration)
	if err != nil {
		return nil, fmt.Errorf("could not create reader: %w", err)
	}
	reader.SetMaxOpenConns(100)
	writer, err := newDatabase(configuration)
	if err != nil {
		return nil, fmt.Errorf("could not create writer: %w", err)
	}
	// only one writer connection as there can only be one
	writer.SetMaxOpenConns(1)
	writer.SetMaxIdleConns(1)

	return &Database{
		reader: reader,
		writer: writer,
	}, nil
}

func newDatabase(configuration config.Configuration) (*sql.DB, error) {
	db, err := sql.Open("sqlite", fmt.Sprintf("%s?_pragma=journal_mode(WAL)", configuration.Database.Filename))
	if err != nil {
		return nil, fmt.Errorf("could not open database %s: %w", configuration.Database, err)
	}

	if _, err := db.Exec(create); err != nil {
		return nil, fmt.Errorf("could not create tables: %w", err)
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
	err1 := db.writer.Close()
	err2 := db.reader.Close()
	return errors.Join(err1, err2)
}
