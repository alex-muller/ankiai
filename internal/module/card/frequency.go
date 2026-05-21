package card

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/alex-muller/ankiai/internal/lib/logger"
)

func newFrequency(repo *CardRepo) frequency {
	return frequency{
		log:  logger.Logger.With("component", "frequency"),
		repo: repo,
	}
}

type frequency struct {
	log  *slog.Logger
	repo *CardRepo
}

func (a *frequency) run(ctx context.Context) {
	words, err := a.repo.FindManyUniqueTargetWordsByStatus(ctx, StatusCreated, 10)
	if err != nil {
		a.log.Error(`find cards by status "created" failed`, slog.String("error", err.Error()))
	}

	err = a.processWords(ctx, words)
	if err != nil {
		a.log.Error(`process words failed`, slog.String("error", err.Error()))
	}
}

func (a *frequency) processWords(ctx context.Context, words []string) error {
	if len(words) == 0 {
		return nil
	}
	rawHtml, err := a.makeRequest(ctx, words, 1990, 2022)
	if err != nil {
		return fmt.Errorf(`make request: %w`, err)
	}

	averages, err := a.getAverages(rawHtml)
	if err != nil {
		return fmt.Errorf(`get averages: %w`, err)
	}

	_ = averages

	return nil
}

func (a *frequency) makeRequest(ctx context.Context, words []string, fromYear, toYear int) (string, error) {
	if len(words) == 0 {
		return ``, nil
	}
	baseURL := "https://books.google.com/ngrams/graph"
	params := url.Values{}
	params.Add("content", strings.Join(words, ","))
	params.Add("year_start", "2000")
	params.Add("year_end", "2022")
	params.Add("corpus", "en")

	fullURL := baseURL + "?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return ``, fmt.Errorf(`new request: %w`, err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return ``, fmt.Errorf(`make request: %w`, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ``, fmt.Errorf(`bad status code: %d`, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ``, fmt.Errorf(`read data: %w`, err)
	}

	return string(body), nil
}

// NgramData represents the required fields from the JSON payload.
// We omit "parent" and "type" as they are not needed for our calculation.
type NgramData struct {
	Ngram      string    `json:"ngram"`
	Timeseries []float64 `json:"timeseries"`
}

func (a *frequency) getAverages(rawHtml string) (map[string]float64, error) {
	// 1. Extract JSON using Regular Expression
	// (?s) allows the dot (.) to match newline characters
	re := regexp.MustCompile(`(?s)<script id="ngrams-data" type="application/json">\s*(.*?)\s*</script>`)
	matches := re.FindStringSubmatch(rawHtml)

	if len(matches) < 2 {
		return nil, nil
	}

	rawJSON := matches[1]

	// 2. Parse the JSON
	var ngrams []NgramData
	if err := json.Unmarshal([]byte(rawJSON), &ngrams); err != nil {
		return nil, fmt.Errorf(`json unmarshal: %w`, err)
	}

	// 3. Calculate averages
	// Using a map to store word -> average mapping
	averages := make(map[string]float64)

	for _, item := range ngrams {
		if len(item.Timeseries) == 0 {
			averages[item.Ngram] = 0
			continue
		}

		var sum float64
		for _, val := range item.Timeseries {
			sum += val
		}

		averages[item.Ngram] = sum / float64(len(item.Timeseries))
	}

	// 4. Output results
	out := make(map[string]float64)

	fmt.Println("Calculated Ngram Averages:")
	for word, avg := range averages {
		out[word] = avg
	}

	return out, nil
}
