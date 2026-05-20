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
	_, err := a.db.NamedExecContext(ctx, `
		INSERT INTO anki_cards (
			word_id, lemma, card_hash, target_word_form, marked_sentence,
			translation, grammar_note, synonyms, part_of_speech,
			definition_en, definition_ru, translation_ru,
			audio_filename, audio_base64, status
		) VALUES (
			:word_id, :lemma, :card_hash, :target_word_form, :marked_sentence,
			:translation, :grammar_note, :synonyms, :part_of_speech,
			:definition_en, :definition_ru, :translation_ru,
			:audio_filename, :audio_base64, :status
		)`, card)
	return err
}
