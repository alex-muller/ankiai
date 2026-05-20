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

func (a Repository) Update(ctx context.Context, w Word) error {
	query := `UPDATE words SET status = :status, raw_json = :raw_json, error_log = :error_log WHERE id = :id`
	_, err := a.db.NamedExecContext(ctx, query, w)
	return err
}

func (a Repository) GetAdded(ctx context.Context) ([]Word, error) {
	query := "SELECT * FROM words WHERE status = $1 LIMIT 1000"
	rows, err := a.db.QueryxContext(ctx, query, StatusAdded)
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
