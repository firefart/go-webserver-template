package database

import (
	"context"
)

type MockDB struct{}

func NewMockDB() *MockDB {
	mockDB := MockDB{}
	return &mockDB
}

// compile time check that struct implements the interface
var _ Interface = (*MockDB)(nil)

func (*MockDB) Close() error { return nil }

func (*MockDB) GetAllDummy(_ context.Context) ([]int64, error) {
	return nil, nil
}

func (*MockDB) InsertDummy(_ context.Context, _ string) (int64, error) {
	return -1, nil
}
