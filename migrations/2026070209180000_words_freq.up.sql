-- +goose Up
-- +goose StatementBegin
alter table words
    add frequency     REAL     default 0  not null;
-- +goose StatementEnd