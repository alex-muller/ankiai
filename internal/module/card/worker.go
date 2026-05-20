package card

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/alex-muller/ankiai/internal/lib/logger"
	"github.com/alex-muller/ankiai/internal/lib/wp"
	"github.com/alex-muller/ankiai/internal/module/word"
)

func NewWorker(wordsRepo *word.Repository, cardRepo *CardRepo) *Worker {
	return &Worker{
		cardRepo:  cardRepo,
		wordsRepo: wordsRepo,
		log:       logger.Logger.With(slog.String("component", "worker")),
	}
}

type Worker struct {
	cardRepo  *CardRepo
	wordsRepo *word.Repository
	log       *slog.Logger
}

func (a Worker) Run(ctx context.Context) {
	ch := make(chan word.Word)

	wg := &sync.WaitGroup{}
	wg.Add(1)

	go func() {
		wg.Done()
		for {
			select {
			case <-ctx.Done():
				close(ch)
				return
			default:
				addedWords, err := a.wordsRepo.GetThousandByStatus(ctx, word.StatusRaw)
				if err != nil {
					a.log.Error(`get words to process`, slog.String("error", err.Error()))
					close(ch)
				}

				if len(addedWords) == 0 {
					time.Sleep(1 * time.Second)
					continue
				}

				for _, addedWord := range addedWords {
					wg.Add(1)
					ch <- addedWord
				}
				wg.Wait()
			}
		}
	}()

	pool := wp.NewWorkerPool(10, 10000)
	pool.Start()

	var i int
	for word_ := range ch {
		i++
		taskID := i
		task := wp.Task{
			ID:      taskID,
			Payload: word_,
			Process: func(ctx context.Context, word_ any) error {
				defer wg.Done()
				w, ok := word_.(word.Word)
				if !ok {
					return errors.New(`invalid word type`)
				}
				return a.processWord(ctx, w)
			},
		}

		if err := pool.Submit(task); err != nil {
			a.log.Error("failed to submit task",
				slog.Int("task_id", taskID),
				slog.String("error", err.Error()))
		}
	}

	a.log.Debug(`task finished`)
}

func (a Worker) processWord(ctx context.Context, word_ word.Word) error {
	resp := GeminiResponse{}

	err := json.Unmarshal([]byte(word_.RawJSON), &resp)
	if err != nil {
		return fmt.Errorf(`json decode raw: %w`, err)
	}

	var cardJson CardJson

	for _, candidate := range resp.Candidates {
		for _, part := range candidate.Content.Parts {
			err = json.Unmarshal([]byte(part.Text), &cardJson)
			if err != nil {
				return fmt.Errorf(`json decode part: %w`, err)
			}
		}
	}

	if cardJson.Lemma == `` {
		return nil
	}

	err = a.processCardJson(ctx, cardJson, word_)
	if err != nil {
		return fmt.Errorf(`process card: %w`, err)
	}

	word_.Status = word.StatusCardsCreated
	err = a.wordsRepo.Update(ctx, word_)
	if err != nil {
		return fmt.Errorf(`update word: %w`, err)
	}

	return nil
}

func (a Worker) processCardJson(ctx context.Context, cardJson CardJson, word_ word.Word) error {
	now := time.Now()
	for _, sens := range cardJson.Senses {
		for _, example := range sens.Examples {
			sum := md5.Sum([]byte(example.MarkedSentence))
			ankiCard := AnkiCard{
				WordID:         word_.ID,
				Lemma:          strings.ToLower(cardJson.Lemma),
				CardHash:       hex.EncodeToString(sum[:]),
				TargetWordForm: strings.ToLower(example.TargetWordForm),
				MarkedSentence: example.MarkedSentence,
				Translation:    example.Translation,
				GrammarNote:    example.GrammarNote,
				Synonyms:       strings.Join(sens.Synonyms, ", "),
				PartOfSpeech:   strings.ToLower(sens.PartOfSpeech),
				DefinitionEn:   sens.DefinitionEn,
				DefinitionRu:   sens.DefinitionRu,
				TranslationRu:  example.Translation,
				AudioFilename:  "",
				AudioBase64:    "",
				Status:         0,
				CreatedAt:      now,
			}

			err := a.cardRepo.Add(ctx, ankiCard)

			a.log.Info(
				`added card`,
				slog.String(`lemma`, ankiCard.Lemma),
				slog.String(`word`, ankiCard.TargetWordForm),
			)

			if err != nil {
				return fmt.Errorf(`add card: %w`, err)
			}
		}
	}

	return nil
}

type GeminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

type CardJson struct {
	Lemma  string `json:"lemma"`
	Senses []struct {
		PartOfSpeech  string   `json:"part_of_speech"`
		TranslationRu string   `json:"translation_ru"`
		DefinitionEn  string   `json:"definition_en"`
		DefinitionRu  string   `json:"definition_ru"`
		Synonyms      []string `json:"synonyms"`
		Examples      []struct {
			GrammarNote    string `json:"grammar_note"`
			TargetWordForm string `json:"target_word_form"`
			MarkedSentence string `json:"marked_sentence"`
			Translation    string `json:"translation"`
		} `json:"examples"`
	} `json:"senses"`
}
