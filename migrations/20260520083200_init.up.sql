CREATE TABLE word_requests
(
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    target_word TEXT UNIQUE NOT NULL,       -- Например: "designated"
    status      TEXT     DEFAULT 'pending', -- Статусы: pending, gemini_done, error
    raw_json    TEXT,                       -- Сырой ответ от Gemini (для бэкапа и дебага)
    error_log   TEXT,                       -- Лог ошибки, если Gemini вернул 429/500
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE anki_cards
(
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    request_id       INTEGER     NOT NULL,          -- Связь с таблицей word_requests
    card_hash        TEXT UNIQUE NOT NULL,          -- Наш Primary Key для Anki (хэш чистого предложения)

    -- Контент карточки
    target_word_form TEXT        NOT NULL,          -- Точная форма из поля target_word_form
    marked_sentence  TEXT        NOT NULL,          -- Предложение со звездочками **word**
    translation      TEXT        NOT NULL,          -- Перевод предложения
    extra_context    TEXT,                          -- Сюда клеим часть речи, синонимы и дефиницию для "изнанки"

    -- Медиа
    audio_filename   TEXT,                          -- Например: "vocab_8a4b.mp3"
    audio_base64     TEXT,                          -- Base64 от Google Cloud TTS

    -- Управление стейт-машиной
    status           TEXT     DEFAULT 'pending_tts',-- Статусы: pending_tts, ready_for_anki, synced, error
    error_log        TEXT,
    created_at       DATETIME DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (request_id) REFERENCES word_requests (id) ON DELETE CASCADE
);