package run

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/alex-muller/ankiai/internal/lib/logger"
	"github.com/alex-muller/ankiai/internal/module/word"
)

func ImportFromFile(ctx context.Context, repository *word.Repository, filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf(`read file: %w`, err)
	}

	// Если это текстовый файл, конвертируем []byte в string
	content := string(data)

	// Attach context to all logs generated within this function
	l := logger.Logger.With(
		slog.String("component", "importer"),
		slog.String("file", filePath),
	)

	l.Info("starting import process")

	words, err := parseAndSanitizeWords(content)
	if err != nil {
		l.Error("failed to process import file", slog.String("error", err.Error()))
		return err
	}

	if len(words) == 0 {
		l.Warn("no valid words found in the file, aborting import")
		return err
	}

	l.Info("file parsed successfully", slog.Int("unique_words_count", len(words)))

	added, err := repository.Add(ctx, words)
	if err != nil {
		return fmt.Errorf(`add words: %w`, err)
	}

	l.Info("import completed successfully for words", slog.Int("unique_words_count", added))

	return nil
}

func parseAndSanitizeWords(text string) ([]string, error) {
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
