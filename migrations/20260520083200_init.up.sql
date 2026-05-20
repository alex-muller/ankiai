-- +goose Up
-- +goose StatementBegin
CREATE TABLE words
(
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    word TEXT UNIQUE NOT NULL,
    status        INTEGER  DEFAULT 0,
    raw_json      TEXT,
    error_log     TEXT,
    created_at    DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE anki_cards
(
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    request_id       INTEGER     NOT NULL,
    lemma            TEXT        NOT NULL,
    card_hash        TEXT UNIQUE NOT NULL,
    target_word_form TEXT        NOT NULL,
    marked_sentence  TEXT        NOT NULL,
    translation      TEXT        NOT NULL,
    extra_context    TEXT,
    audio_filename   TEXT,
    audio_base64     TEXT,
    status           INTEGER  DEFAULT 0,
    error_log        TEXT,
    created_at       DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (request_id) REFERENCES word_requests (id) ON DELETE CASCADE
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS anki_cards;
DROP TABLE IF EXISTS word_requests;
-- +goose StatementEnd