-- +goose Up
-- +goose StatementBegin
CREATE TABLE words
(
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    word       TEXT UNIQUE NOT NULL,
    status     INTEGER              DEFAULT 0,
    raw_json   TEXT        NOT NULL DEFAULT '',
    error_log  TEXT        NOT NULL DEFAULT '',
    created_at DATETIME             DEFAULT CURRENT_TIMESTAMP
);

create table notes
(
    id               INTEGER
        primary key autoincrement,
    word_id          INTEGER             not null
        references words
            on delete cascade,
    anki_note_id     integer  default 0  not null,
    lemma            TEXT                not null,
    card_hash        TEXT                not null
        unique,
    target_word_form TEXT                not null,
    marked_sentence  TEXT                not null,
    translation      TEXT                not null,
    grammar_note     TEXT     default '' not null,
    synonyms         TEXT     default '' not null,
    part_of_speech   TEXT     default '' not null,
    definition_en    TEXT     default '' not null,
    definition_ru    TEXT     default '' not null,
    translation_ru   TEXT     default '' not null,
    audio_filename   TEXT     default '' not null,
    audio_base64     TEXT     default '' not null,
    status           INTEGER  default 0,
    frequency        REAL     default 0  not null,
    created_at       DATETIME default CURRENT_TIMESTAMP
);

create index anki_cards_status_target_word_form_index
    on notes (status, target_word_form);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS notes;
DROP TABLE IF EXISTS words;
-- +goose StatementEnd