package database

import (
	"context"
)

type Interface interface {
	Close() error
	InsertDummy(ctx context.Context, name string) (int64, error)
	GetAllDummy(ctx context.Context) ([]int64, error)
}

// compile time check that struct implements the interface
var _ Interface = (*Database)(nil)

func (db *Database) GetAllDummy(ctx context.Context) ([]int64, error) {
	dummies, err := db.reader.GetAllDummy(ctx)
	if err != nil {
		return nil, err
	}
	ids := make([]int64, len(dummies))
	for i, dummy := range dummies {
		ids[i] = dummy.ID
	}
	return ids, nil
}

func (db *Database) InsertDummy(ctx context.Context, name string) (int64, error) {
	dummy, err := db.writer.InsertDummy(ctx, name)
	if err != nil {
		return -1, err
	}
	return dummy.ID, nil
}
