package card

import (
	"context"

	"github.com/jmoiron/sqlx"
)

func NewCardRepo(db *sqlx.DB) *CardRepo {
	return &CardRepo{db: db}
}

type CardRepo struct {
	db *sqlx.DB
}

func (a CardRepo) Add(ctx context.Context, card AnkiCard) error {
	const query = `
		INSERT INTO anki_cards (word_id, lemma, card_hash, target_word_form, marked_sentence, translation, audio_filename, audio_base64, status)
		VALUES (:word_id, :lemma, :card_hash, :target_word_form, :marked_sentence, :translation, :audio_filename, :audio_base64, :status)
		ON CONFLICT(card_hash) DO NOTHING`

	_, err := a.db.NamedExecContext(ctx, query, card)
	return err
}
