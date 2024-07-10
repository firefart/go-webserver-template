package database

type MockDB struct{}

func NewMockDB() *MockDB {
	mockDB := MockDB{}
	return &mockDB
}

// compile time check that struct implements the interface
var _ DatabaseInterface = (*MockDB)(nil)

func (db *MockDB) Close() error { return nil }
