package importer

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/alex-muller/ankiai/internal/lib/logger"
	"github.com/alex-muller/ankiai/internal/module/word"
)

func NewService(repository *word.Repository) *Service {
	return &Service{
		repository: repository,
		log: logger.Logger.With(
			slog.String("component", "importer"),
		),
	}
}

type Service struct {
	repository *word.Repository
	log        *slog.Logger
}

func (a Service) ImportFromFile(ctx context.Context, filePath string) ([]string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf(`read file: %w`, err)
	}

	text := string(data)
	return a.importText(ctx, text)
}

func (a Service) importText(ctx context.Context, text string) ([]string, error) {
	words, err := prepare(text)
	if err != nil {
		return nil, fmt.Errorf(`prepare words: %w`, err)
	}

	if len(words) == 0 {
		return nil, nil
	}

	return a.repository.Add(ctx, words)
}

func prepare(text string) ([]string, error) {
	// Using a map with an empty struct as a memory-efficient Set for deduplication
	uniqueSet := make(map[string]struct{})
	var result []string

	// Split by comma to handle multiple words on a single line
	parts := strings.Split(text, ",")

	for _, part := range parts {
		word := strings.Join(strings.Fields(part), " ")

		// 2. Normalize: convert to lowercase
		word = strings.ToLower(word)

		// 3. Validate: skip empty strings
		if word == "" {
			continue
		}

		// 4. Deduplicate: check if word is already in our Set
		if _, exists := uniqueSet[word]; !exists {
			uniqueSet[word] = struct{}{}
			result = append(result, word)
		}
	}

	return result, nil
}
