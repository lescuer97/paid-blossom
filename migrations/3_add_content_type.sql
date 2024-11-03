-- +goose Up
ALTER TABLE blobs ADD content_type TEXT;


-- +goose Down
ALTER TABLE blobs DROP COLUMN content_type;
