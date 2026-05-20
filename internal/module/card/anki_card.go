package card

import "time"

type Status int

const (
	StatusCreated Status = 0
)

type AnkiCard struct {
	ID             int       `db:"id"`
	WordID         int       `db:"word_id"`
	Lemma          string    `db:"lemma"`
	CardHash       string    `db:"card_hash"`
	TargetWordForm string    `db:"target_word_form"`
	MarkedSentence string    `db:"marked_sentence"`
	Translation    string    `db:"translation"`
	GrammarNote    string    `db:"grammar_note"`
	Synonyms       string    `db:"synonyms"`
	PartOfSpeech   string    `db:"part_of_speech"`
	DefinitionEn   string    `db:"definition_en"`
	DefinitionRu   string    `db:"definition_ru"`
	TranslationRu  string    `db:"translation_ru"`
	AudioFilename  string    `db:"audio_filename"`
	AudioBase64    string    `db:"audio_base64"`
	Status         Status    `db:"status"`
	Frequency      float64   `db:"frequency"`
	CreatedAt      time.Time `db:"created_at"`
}
