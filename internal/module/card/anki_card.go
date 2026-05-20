package card

import "time"

type AnkiCard struct {
	ID             int       `db:"id"`
	RequestID      int       `db:"request_id"`
	Lemma          string    `db:"lemma"`
	CardHash       string    `db:"card_hash"`
	TargetWordForm string    `db:"target_word_form"`
	MarkedSentence string    `db:"marked_sentence"`
	Translation    string    `db:"translation"`
	ExtraContext   *string   `db:"extra_context"`
	AudioFilename  *string   `db:"audio_filename"`
	AudioBase64    *string   `db:"audio_base64"`
	Status         string    `db:"status"`
	ErrorLog       *string   `db:"error_log"`
	CreatedAt      time.Time `db:"created_at"`
}
