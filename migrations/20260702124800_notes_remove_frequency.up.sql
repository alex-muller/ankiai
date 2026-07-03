-- +goose Up
-- +goose StatementBegin
ALTER TABLE notes DROP COLUMN frequency;
-- +goose StatementEnd
