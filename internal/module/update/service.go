package update

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/alex-muller/ankiai/internal/module/frequency"
	"github.com/alex-muller/ankiai/internal/module/lexicographer"
	"github.com/alex-muller/ankiai/internal/module/notes"
	word2 "github.com/alex-muller/ankiai/internal/module/word"
)

func New(
	freq frequency.FrequencyService,
	wordsRepo *word2.Repository,
	notesRepo *notes.Repo,
	lex *lexicographer.Service,
) Service {
	return Service{
		freq:      freq,
		wordsRepo: wordsRepo,
		notesRepo: notesRepo,
		lex:       lex,
	}
}

type Service struct {
	freq      frequency.FrequencyService
	wordsRepo *word2.Repository
	notesRepo *notes.Repo
	lex       *lexicographer.Service
}

func (a Service) Run(ctx context.Context) error {
	err := a.updateFreq(ctx)
	if err != nil {
		return fmt.Errorf(`update freq: %w`, err)
	}

	err = a.updatePlTranslate(ctx)
	if err != nil {
		return fmt.Errorf(`update pl translate: %w`, err)
	}

	return nil
}

func (a Service) updatePlTranslate(ctx context.Context) error {
	notes, err := a.notesRepo.FindManyForPolishUpdate(ctx, 0)
	if err != nil {
		return fmt.Errorf(`find notes: %w`, err)
	}

	total := len(notes)
	var count int

	for {
		notes, err = a.notesRepo.FindManyForPolishUpdate(ctx, 1)
		if err != nil {
			return fmt.Errorf(`find note: %w`, err)
		}

		if len(notes) == 0 {
			break
		}

		note := notes[0]

		translate, err := a.lex.GetPlTranslate(ctx, note.TargetWordForm, note.GetSentenceEn(), note.Translation, note.DefinitionEn, note.DefinitionRu)
		if err != nil {
			return fmt.Errorf(`get pl translate: %w`, err)
		}

		_ = translate

		count++

		fmt.Printf("\r Got PL translate %d of %d", count, total)
	}

	return nil
}

func (a Service) updateFreq(ctx context.Context) error {
	err := a.wordsRepo.UpdateWordsForFrequencyGet(ctx)
	if err != nil {
		return fmt.Errorf(`update words: %w`, err)
	}

	for {
		left, err := a.wordsRepo.CountByStatus(ctx, word2.StatusTmpFrequencyRequired)
		if err != nil {
			return fmt.Errorf(`count of frequency left: %w`, err)
		}
		fmt.Printf("\rProcess frequency. Left: %d\n", left)

		if left == 0 {
			break
		}

		phrases, err := a.wordsRepo.FindManyUniqueWordsByStatus(ctx, word2.StatusTmpFrequencyRequired, 10)
		if err != nil {
			return fmt.Errorf(`find words: %w`, err)
		}

		err = a.freq.RunOnceOnPhrases(ctx, phrases)

		if err != nil {
			if strings.Contains(err.Error(), `429`) {
				fmt.Println(`Frequency limit exceeded. Sleep`)
				time.Sleep(time.Minute * 10)
				continue
			}
			return fmt.Errorf(`run phrases: %w`, err)
		}
	}

	return nil
}
