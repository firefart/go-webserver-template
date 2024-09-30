-- name: GetAllDummy :many
SELECT *
FROM dummy
order by id;

-- name: InsertDummy :one
INSERT INTO dummy(name)
VALUES(?)
RETURNING *;
