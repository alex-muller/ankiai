package card

import "time"

type WordRequest struct {
	ID           int       `db:"id"`
	ImportedWord string    `db:"imported_word"`
	Status       string    `db:"status"`
	RawJSON      *string   `db:"raw_json"` // Pointer, т.к. может быть NULL в начале
	ErrorLog     *string   `db:"error_log"`
	CreatedAt    time.Time `db:"created_at"`
}
