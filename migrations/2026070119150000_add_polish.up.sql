-- +goose Up
-- +goose StatementBegin
alter table notes
    add definition_pl text default '' not null;

alter table notes
    add translation_pl text default '' not null;

alter table notes
    add audio_filename_pl text default '' not null;

alter table notes
    add audio_base64_pl text default '' not null;
-- +goose StatementEnd