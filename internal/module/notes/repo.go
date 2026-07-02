package notes

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
)

func NewRepo(db *sqlx.DB) *Repo {
	return &Repo{db: db}
}

type Repo struct {
	db *sqlx.DB
}

func (a Repo) Add(ctx context.Context, card Note) error {
	_, err := a.db.NamedExecContext(ctx, `
		INSERT INTO notes (
			word_id, lemma, card_hash, target_word_form, marked_sentence,
			translation, grammar_note, synonyms, part_of_speech,
			definition_en, definition_ru, translation_ru, definition_pl, translation_pl,
			audio_filename, audio_base64, status
		) VALUES (
			:word_id, :lemma, :card_hash, :target_word_form, :marked_sentence,
			:translation, :grammar_note, :synonyms, :part_of_speech,
			:definition_en, :definition_ru, :translation_ru, :definition_pl, :translation_pl,
			:audio_filename, :audio_base64, :status
		)`, card)
	return err
}

func (a Repo) FindManyUniqueTargetWordsByStatus(ctx context.Context, status Status, limit int) ([]string, error) {
	var out = make([]string, 0, limit)

	err := a.db.SelectContext(ctx, &out, "SELECT DISTINCT target_word_form FROM notes WHERE status = ? LIMIT ?", status, limit)
	if err != nil {
		return nil, fmt.Errorf(`query: %w`, err)
	}

	return out, err
}

// Deprecated.
func (a Repo) FindManyUniqueLemmaWordsByStatus(ctx context.Context, status Status, limit int) ([]string, error) {
	var out = make([]string, 0, limit)

	err := a.db.SelectContext(ctx, &out, "SELECT DISTINCT lemma FROM notes WHERE status = ? LIMIT ?", status, limit)
	if err != nil {
		return nil, fmt.Errorf(`query: %w`, err)
	}

	return out, err
}

func (a Repo) GetManyByStatus(ctx context.Context, status Status, limit int) ([]Note, error) {
	var limitStr = ``
	if limit > 0 {
		limitStr = fmt.Sprintf(` LIMIT %d`, limit)
	}

	query := fmt.Sprintf("SELECT * FROM notes WHERE status = $1 ORDER BY frequency DESC%s", limitStr)
	rows, err := a.db.QueryxContext(ctx, query, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var notes []Note
	for rows.Next() {
		var word Note
		if err := rows.StructScan(&word); err != nil {
			return nil, fmt.Errorf(`scan: %w`, err)
		}
		notes = append(notes, word)
	}
	return notes, nil
}

func (a Repo) UpdateFrequenciesByTargetWord(ctx context.Context, words map[string]float64) error {
	if len(words) == 0 {
		return nil
	}

	const query = `
		UPDATE notes
		SET frequency = ?, status = ?
		WHERE target_word_form = ?
	`

	for word, freq := range words {
		_, err := a.db.ExecContext(ctx, query, freq, GenerateAudioPending, word)
		if err != nil {
			return fmt.Errorf(`update frequency for "%s": %w`, word, err)
		}
	}

	return nil
}

func (a Repo) UpdateFrequenciesByLemma(ctx context.Context, words map[string]float64) error {
	if len(words) == 0 {
		return nil
	}

	const query = `
		UPDATE notes
		SET frequency = ?, status = ?
		WHERE lemma = ?
	`

	for word, freq := range words {
		_, err := a.db.ExecContext(ctx, query, freq, GenerateAudioPending, word)
		if err != nil {
			return fmt.Errorf(`update frequency for "%s": %w`, word, err)
		}
	}

	return nil
}

func (a Repo) AddAudio(ctx context.Context, noteId int64, audioContent, audioFilename string) error {
	q := "UPDATE notes SET audio_filename = ?, audio_base64 = ?, status = ? WHERE id = ?"

	_, err := a.db.ExecContext(ctx, q, audioFilename, audioContent, ExportPending, noteId)
	return err
}

func (a Repo) AddAudioPl(ctx context.Context, noteId int64, audioContent, audioFilename string) error {
	q := "UPDATE notes SET audio_filename_pl = ?, audio_base64_pl = ?, status = ? WHERE id = ?"

	_, err := a.db.ExecContext(ctx, q, audioFilename, audioContent, ExportPending, noteId)
	return err
}

func (a Repo) SetAsExported(ctx context.Context, noteId, ankiNoteId int64, status Status) error {
	q := "UPDATE notes SET status = ?, anki_note_id = ? WHERE id = ?"
	_, err := a.db.ExecContext(ctx, q, status, ankiNoteId, noteId)
	return err
}
