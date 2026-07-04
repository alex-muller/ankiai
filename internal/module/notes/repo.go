package notes

import (
	"context"
	"fmt"
	"time"

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
			audio_filename, audio_base64, created_at, updated_at, status
		) VALUES (
			:word_id, :lemma, :card_hash, :target_word_form, :marked_sentence,
			:translation, :grammar_note, :synonyms, :part_of_speech,
			:definition_en, :definition_ru, :translation_ru, :definition_pl, :translation_pl,
			:audio_filename, :audio_base64, :created_at, :updated_at, :status
		)`, card)
	return err
}

func (a Repo) Update(ctx context.Context, card Note) error {
	_, err := a.db.NamedExecContext(ctx, `
		UPDATE notes SET 
			word_id = :word_id, 
			lemma = :lemma,
			card_hash = :card_hash,
			target_word_form = :target_word_form,
			marked_sentence = :marked_sentence,
			translation = :translation,
			grammar_note = :grammar_note,
			synonyms = :synonyms,
			part_of_speech = :part_of_speech,
			definition_en = :definition_en, 
			definition_ru = :definition_ru,
			translation_ru = :translation_ru,
			definition_pl = :definition_pl,
			translation_pl = :translation_pl,
			audio_filename = :audio_filename,
			audio_base64 = :audio_base64,
			audio_filename_pl = :audio_filename_pl,
			audio_base64_pl = :audio_base64_pl,
			created_at = :created_at,
			updated_at = :updated_at,
			status = :status
		WHERE id = :id`, card)
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

func (a Repo) FindManyForAnkiUpdate(ctx context.Context, limit int) ([]Note, error) {
	var limitStr = ``
	if limit > 0 {
		limitStr = fmt.Sprintf(` LIMIT %d`, limit)
	}

	query := fmt.Sprintf(`
	SELECT n.*, w.frequency FROM notes n 
		LEFT JOIN words w ON n.word_id = w.id 
	WHERE n.exported_at < n.updated_at 
	AND n.status = ?
	ORDER BY w.frequency DESC%s`,
		limitStr)
	rows, err := a.db.QueryxContext(ctx, query, Exported)
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

func (a Repo) FindManyForPolishTranslateUpdate(ctx context.Context, limit int) ([]Note, error) {
	var limitStr = ``
	if limit > 0 {
		limitStr = fmt.Sprintf(` LIMIT %d`, limit)
	}

	query := fmt.Sprintf(`
	SELECT n.*, w.frequency FROM notes n 
		LEFT JOIN words w ON n.word_id = w.id 
	WHERE n.translation_pl = '' 
	ORDER BY w.frequency DESC%s`,
		limitStr)
	rows, err := a.db.QueryxContext(ctx, query)
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

func (a Repo) FindManyForPolishTTSUpdate(ctx context.Context, limit int) ([]Note, error) {
	var limitStr = ``
	if limit > 0 {
		limitStr = fmt.Sprintf(` LIMIT %d`, limit)
	}

	query := fmt.Sprintf(`
	SELECT n.*, w.frequency FROM notes n 
		LEFT JOIN words w ON n.word_id = w.id 
	WHERE n.audio_filename_pl = ''
		AND n.audio_base64_pl = ''
		AND n.translation_pl <> ''
	ORDER BY w.frequency DESC%s`,
		limitStr)
	rows, err := a.db.QueryxContext(ctx, query)
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

func (a Repo) GetManyByStatus(ctx context.Context, status Status, limit int) ([]Note, error) {
	var limitStr = ``
	if limit > 0 {
		limitStr = fmt.Sprintf(` LIMIT %d`, limit)
	}

	query := fmt.Sprintf("SELECT n.*, w.frequency FROM notes n LEFT JOIN words w ON n.word_id = w.id WHERE n.status = $1 ORDER BY w.frequency DESC%s", limitStr)
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

func (a Repo) AddAudio(ctx context.Context, noteId int64, audioContent, audioFilename string) error {
	q := "UPDATE notes SET audio_filename = ?, audio_base64 = ?, status = ?, updated_at = ?  WHERE id = ?"

	_, err := a.db.ExecContext(ctx, q, audioFilename, audioContent, ExportPending, time.Now().UTC(), noteId)
	return err
}

func (a Repo) AddAudioPl(ctx context.Context, noteId int64, audioContent, audioFilename string) error {
	q := "UPDATE notes SET audio_filename_pl = ?, audio_base64_pl = ?, updated_at = ? WHERE id = ?"

	_, err := a.db.ExecContext(ctx, q, audioFilename, audioContent, time.Now().UTC(), noteId)
	return err
}

func (a Repo) SetAsExported(ctx context.Context, noteId, ankiNoteId int64, status Status) error {
	q := "UPDATE notes SET status = ?, anki_note_id = ?, exported_at = ? WHERE id = ?"
	_, err := a.db.ExecContext(ctx, q, status, ankiNoteId, time.Now().UTC(), noteId)
	return err
}
