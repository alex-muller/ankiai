package frequency

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
	"time"

	"github.com/alex-muller/ankiai/internal/lib"
	"github.com/alex-muller/ankiai/internal/lib/logger"
	"github.com/alex-muller/ankiai/internal/module/notes"
	word2 "github.com/alex-muller/ankiai/internal/module/word"
)

func NewFrequency(repo *notes.Repo, wordsRepo *word2.Repository) FrequencyService {
	return FrequencyService{
		log:       logger.Logger.With("component", "frequency"),
		notesRepo: repo,
		wordsRepo: wordsRepo,
	}
}

type FrequencyService struct {
	log       *slog.Logger
	notesRepo *notes.Repo
	wordsRepo *word2.Repository
}

func (a *FrequencyService) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			err := a.runOnceOnWords(ctx)
			if err != nil {
				a.log.Error(`process words failed`, slog.String("error", err.Error()))
			}

			time.Sleep(time.Second)
		}
	}
}

func (a *FrequencyService) runOnceOnWords(ctx context.Context) error {
	phrases, err := a.wordsRepo.FindManyUniqueWordsByStatus(ctx, word2.StatusNew, 10)
	if err != nil {
		a.log.Error(`find cards by status "created" failed`, slog.String("error", err.Error()))
		return err
	}

	return a.RunOnceOnPhrases(ctx, phrases)
}

func (a *FrequencyService) RunOnceOnPhrases(ctx context.Context, phrases []string) error {
	if len(phrases) == 0 {
		return nil
	}

	wordsMap := a.cleanWords(phrases)

	var words []string
	for w := range wordsMap {
		words = append(words, w)
	}

	rawHtml, err := a.makeRequest(ctx, words, 1990, 2022)
	if err != nil {
		return fmt.Errorf(`make request: %w`, err)
	}

	averagesForEachWord, err := a.getAverages(rawHtml)
	if err != nil {
		return fmt.Errorf(`get averages: %w`, err)
	}

	if len(averagesForEachWord) == 0 {
		return fmt.Errorf(`can't calculate averages`)
	}

	lemmaMap := make(map[string]float64)

	for _, p := range phrases {
		lemmaMap[p] = 0
	}

	for _, word := range words {
		freq := averagesForEachWord[word]

		for _, phr := range wordsMap[word] {
			if currentFreq := lemmaMap[phr]; currentFreq > freq || currentFreq == 0 {
				lemmaMap[phr] = freq
			}
		}
	}

	err = a.wordsRepo.UpdateFrequenciesByPhrase(ctx, lemmaMap)
	if err != nil {
		return fmt.Errorf(`update frequencies: %w`, err)
	}

	fmt.Println(fmt.Sprintf("Calculated Ngram Averages: %v", words))

	return nil
}

func (a *FrequencyService) makeRequest(ctx context.Context, words []string, fromYear, toYear int) (string, error) {
	if len(words) == 0 {
		return ``, nil
	}
	baseURL := "https://books.google.com/ngrams/graph"
	params := url.Values{}
	params.Add("content", strings.Join(words, ","))
	params.Add("year_start", fmt.Sprintf("%d", fromYear))
	params.Add("year_end", fmt.Sprintf("%d", toYear))
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

func (a *FrequencyService) getAverages(rawHtml string) (map[string]float64, error) {
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

	for word, avg := range averages {
		word = strings.ReplaceAll(word, " - ", "-")
		out[word] = avg
	}

	return out, nil
}

type phrase = string
type word = string

func (a *FrequencyService) cleanWords(phraseList []string) map[word][]phrase {
	var m = make(map[word][]phrase)

	for _, phr := range phraseList {
		for _, wrd := range lib.CleanPhraseAndSplit(phr) {
			m[wrd] = append(m[wrd], phr)
		}
	}

	return m
}
