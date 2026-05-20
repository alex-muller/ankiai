package word

import (
	"context"
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
