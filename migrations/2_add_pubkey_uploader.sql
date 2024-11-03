-- +goose Up
ALTER TABLE blobs ADD pubkey TEXT;


-- +goose Down
ALTER TABLE blobs DROP COLUMN pubkey;

