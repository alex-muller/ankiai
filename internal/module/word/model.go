package word

import "time"

type Status int

const (
	StatusNew            Status = 0
	StatusAddedFrequency Status = 1
	StatusRaw            Status = 2
	StatusCardsCreated   Status = 3

	StatusTmpFrequencyRequired Status = 101
)

type Word struct {
	ID        int       `db:"id"`
	Word      string    `db:"word"`
	Status    Status    `db:"status"`
	RawJSON   string    `db:"raw_json"`
	Frequency float64   `db:"frequency"`
	ErrorLog  string    `db:"error_log"`
	CreatedAt time.Time `db:"created_at"`
}
