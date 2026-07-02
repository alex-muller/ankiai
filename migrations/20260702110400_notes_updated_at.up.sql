-- +goose Up
-- +goose StatementBegin
ALTER TABLE notes ADD COLUMN updated_at DATETIME default NULL;
ALTER TABLE notes ADD COLUMN exported_at DATETIME default NULL;

UPDATE notes SET updated_at = created_at;
UPDATE notes SET exported_at = created_at;
-- +goose StatementEnd
