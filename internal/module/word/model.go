package word

import "time"

type Status int

const (
	StatusNew          Status = 0
	StatusRaw          Status = 1
	StatusCardsCreated Status = 2
)

type Word struct {
	ID        int       `db:"id"`
	Word      string    `db:"word"`
	Status    Status    `db:"status"`
	RawJSON   string    `db:"raw_json"`
	ErrorLog  string    `db:"error_log"`
	CreatedAt time.Time `db:"created_at"`
}
