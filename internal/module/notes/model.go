package notes

import (
	"strings"
	"time"
)

type Status int

const (
	GenerateAudioPending Status = 0
	ExportPending        Status = 1
	Exported             Status = 2
)

type Note struct {
	ID              int64      `db:"id"`
	AnkiNoteID      int64      `db:"anki_note_id"`
	WordID          int        `db:"word_id"`
	Lemma           string     `db:"lemma"`
	CardHash        string     `db:"card_hash"`
	TargetWordForm  string     `db:"target_word_form"`
	MarkedSentence  string     `db:"marked_sentence"`
	Translation     string     `db:"translation"`
	GrammarNote     string     `db:"grammar_note"`
	Synonyms        string     `db:"synonyms"`
	PartOfSpeech    string     `db:"part_of_speech"`
	DefinitionEn    string     `db:"definition_en"`
	DefinitionRu    string     `db:"definition_ru"`
	TranslationRu   string     `db:"translation_ru"`
	DefinitionPl    string     `db:"definition_pl"`
	TranslationPl   string     `db:"translation_pl"`
	AudioFilename   string     `db:"audio_filename"`
	AudioFilenamePl string     `db:"audio_filename_pl"`
	AudioBase64     string     `db:"audio_base64"`
	AudioBase64Pl   string     `db:"audio_base64_pl"`
	Status          Status     `db:"status"`
	Frequency       float64    `db:"frequency"`
	CreatedAt       time.Time  `db:"created_at"`
	UpdatedAt       time.Time  `db:"updated_at"`
	ExportedAt      *time.Time `db:"exported_at"`
}

func (a Note) GetSentenceEn() string {
	return strings.ReplaceAll(a.MarkedSentence, "**", "")
}
