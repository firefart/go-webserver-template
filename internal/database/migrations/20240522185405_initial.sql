-- +goose Up
-- +goose StatementBegin
CREATE TABLE dummy
(
    id      INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    name    TEXT    NOT NULL UNIQUE,
    updated DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_name ON dummy (name);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX idx_name;
DROP TABLE dummy;
-- +goose StatementEnd
