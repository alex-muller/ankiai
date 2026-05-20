package word

import "time"

type Word struct {
	ID        int       `db:"id"`
	Word      string    `db:"word"`
	Status    string    `db:"status"`
	RawJSON   *string   `db:"raw_json"` // Pointer, т.к. может быть NULL в начале
	ErrorLog  *string   `db:"error_log"`
	CreatedAt time.Time `db:"created_at"`
}
