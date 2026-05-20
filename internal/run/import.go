package run

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/alex-muller/ankiai/internal/lib/logger"
	"github.com/jmoiron/sqlx"
)

func RunImport(db *sqlx.DB, filePath string) error {
	// Attach context to all logs generated within this function
	l := logger.Logger.With(
		slog.String("component", "importer"),
		slog.String("file", filePath),
	)

	l.Info("starting import process")

	words, err := parseAndSanitizeWords(filePath)
	if err != nil {
		l.Error("failed to process import file", slog.String("error", err.Error()))
		return err
	}

	if len(words) == 0 {
		l.Warn("no valid words found in the file, aborting import")
		return err
	}

	l.Info("file parsed successfully", slog.Int("unique_words_count", len(words)))

	// TODO: Pass the 'words' slice to the repository
	// err = repository.InsertWords(words)
	// if err != nil {
	// 	l.Error("failed to save words to database", slog.String("error", err.Error()))
	//  return
	// }

	l.Info("import completed successfully")

	return nil
}

func parseAndSanitizeWords(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("could not open file: %w", err)
	}
	defer file.Close()

	// Using a map with an empty struct as a memory-efficient Set for deduplication
	uniqueSet := make(map[string]struct{})
	var result []string

	scanner := bufio.NewScanner(file)

	// Read file line by line
	for scanner.Scan() {
		line := scanner.Text()

		// Split by comma to handle multiple words on a single line
		parts := strings.Split(line, ",")

		for _, part := range parts {
			// 1. Collapse multiple spaces/tabs into a single space
			// strings.Fields splits by any whitespace, strings.Join connects with " "
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
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file stream: %w", err)
	}

	return result, nil
}
