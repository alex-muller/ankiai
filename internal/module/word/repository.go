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

func (a Repository) FindManyUniqueWordsByStatus(ctx context.Context, status Status, limit int) ([]string, error) {
	var out = make([]string, 0, limit)

	err := a.db.SelectContext(ctx, &out, "SELECT DISTINCT word FROM words WHERE status = ? LIMIT ?", status, limit)
	if err != nil {
		return nil, fmt.Errorf(`query: %w`, err)
	}

	return out, err
}

func (a Repository) UpdateFrequenciesByPhrase(ctx context.Context, words map[string]float64) error {
	if len(words) == 0 {
		return nil
	}

	const query = `
		UPDATE words
		SET frequency = ?, status = ?
		WHERE word = ?
	`

	for word, freq := range words {
		_, err := a.db.ExecContext(ctx, query, freq, StatusAddedFrequency, word)
		if err != nil {
			return fmt.Errorf(`update frequency for "%s": %w`, word, err)
		}
	}

	return nil
}

func (a Repository) Update(ctx context.Context, w Word) error {
	query := `UPDATE words SET status = :status, raw_json = :raw_json, error_log = :error_log WHERE id = :id`
	_, err := a.db.NamedExecContext(ctx, query, w)
	return err
}

func (a Repository) GetForExamples(ctx context.Context, status Status) ([]Word, error) {
	query := "SELECT * FROM words WHERE status = $1 ORDER BY frequency DESC"
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
