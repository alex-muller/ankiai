package word

import (
	"context"
	"fmt"

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

func (a Repository) Add(ctx context.Context, words []string) ([]string, error) {
	var output = make([]string, 0, len(words))
	query := "INSERT OR IGNORE INTO words (word) VALUES ($1)"

	for _, word := range words {
		res, err := a.db.ExecContext(ctx, query, word)
		if err != nil {
			return nil, fmt.Errorf(`insert: %w`, err)
		}

		if n, _ := res.RowsAffected(); n == 1 {
			output = append(output, word)
		}
	}

	return output, nil
}

func (a Repository) Update(ctx context.Context, w Word) error {
	query := `UPDATE words SET status = :status, raw_json = :raw_json, error_log = :error_log WHERE id = :id`
	_, err := a.db.NamedExecContext(ctx, query, w)
	return err
}

func (a Repository) GetByStatus(ctx context.Context, status Status) ([]Word, error) {
	query := "SELECT * FROM words WHERE status = $1 ORDER BY id ASC"
	rows, err := a.db.QueryxContext(ctx, query, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var words []Word
	for rows.Next() {
		var word Word
		if err := rows.StructScan(&word); err != nil {
			return nil, err
		}
		words = append(words, word)
	}
	return words, nil
}
