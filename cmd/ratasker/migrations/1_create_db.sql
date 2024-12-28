-- +goose Up
CREATE TABLE "blobs" (
	sha256 blob NOT NULL,
	size integer NOT NULL,
	path text NOT NULL,
	created_at integer NOT NULL
);



-- +goose Down
DROP TABLE IF EXISTS blobs;

