-- +goose Up
CREATE TABLE IF NOT EXISTS counter_table(
    keyset_id TEXT PRIMARY KEY, 
    counter integer NOT NULL
);


-- +goose Down
DROP TABLE IF EXISTS counter_table;
