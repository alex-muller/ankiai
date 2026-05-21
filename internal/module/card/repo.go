package card

import (
	"context"
	"fmt"

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

func (a CardRepo) FindManyUniqueTargetWordsByStatus(ctx context.Context, status Status, limit int) ([]string, error) {
	var out = make([]string, 0, limit)

	err := a.db.SelectContext(ctx, &out, "SELECT DISTINCT target_word_form FROM anki_cards WHERE status = ? LIMIT ?", status, limit)
	if err != nil {
		return nil, fmt.Errorf(`query: %w`, err)
	}

	return out, err
}

func (a CardRepo) UpdateFrequencies(ctx context.Context, words map[string]float64) error {
	if len(words) == 0 {
		return nil
	}

	const query = `
		UPDATE anki_cards
		SET frequency = ?, status = ?
		WHERE target_word_form = ?
	`

	for word, freq := range words {
		_, err := a.db.ExecContext(ctx, query, freq, StatusFrequencyAdded, word)
		if err != nil {
			return fmt.Errorf(`update frequency for "%s": %w`, word, err)
		}
	}

	return nil
}
