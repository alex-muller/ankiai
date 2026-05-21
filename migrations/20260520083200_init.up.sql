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

CREATE TABLE anki_cards
(
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    word_id          INTEGER     NOT NULL,
    lemma            TEXT        NOT NULL,
    card_hash        TEXT UNIQUE NOT NULL,
    target_word_form TEXT        NOT NULL,
    marked_sentence  TEXT        NOT NULL,
    translation      TEXT        NOT NULL,
    grammar_note     TEXT        NOT NULL DEFAULT '',
    synonyms         TEXT        NOT NULL DEFAULT '',
    part_of_speech   TEXT        NOT NULL DEFAULT '',
    definition_en    TEXT        NOT NULL DEFAULT '',
    definition_ru    TEXT        NOT NULL DEFAULT '',
    translation_ru   TEXT        NOT NULL DEFAULT '',
    audio_filename   TEXT        NOT NULL DEFAULT '',
    audio_base64     TEXT        NOT NULL DEFAULT '',
    status           INTEGER     NOT NULL DEFAULT 0,
    frequency        REAL        NOT NULL DEFAULT 0,
    created_at       DATETIME             DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (word_id) REFERENCES words (id) ON DELETE CASCADE
);

create index anki_cards_status_target_word_form_index
    on anki_cards (status, target_word_form);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS anki_cards;
DROP TABLE IF EXISTS words;
-- +goose StatementEnd