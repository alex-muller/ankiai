package word

import (
	"context"
	"iter"
	"strings"

	"github.com/jmoiron/sqlx"
)

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{
		db: db,
	}
}

type Repository struct {
	db *sqlx.DB
}

func (a Repository) Add(ctx context.Context, words []string) (int, error) {
	placeholders := make([]string, len(words))
	args := make([]any, len(words))
	for i, w := range words {
		placeholders[i] = "(?)"
		args[i] = w
	}
	query := "INSERT OR IGNORE INTO words (word) VALUES " + strings.Join(placeholders, ", ")
	res, err := a.db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	n, err := res.RowsAffected()
	return int(n), err
}

func (a Repository) GetAdded(ctx context.Context) iter.Seq2[Word, error] {
	return func(yield func(Word, error) bool) {
		query := "SELECT * FROM words WHERE status = $1"
		rows, err := a.db.QueryxContext(ctx, query, StatusAdded)
		if err != nil {
			yield(Word{}, err)
			return
		}
		defer rows.Close()

		for rows.Next() {
			var w Word
			if err := rows.StructScan(&w); err != nil {
				yield(Word{}, err)
				return
			}
			if !yield(w, nil) {
				return
			}
		}

		if err := rows.Err(); err != nil {
			yield(Word{}, err)
		}
	}
}
