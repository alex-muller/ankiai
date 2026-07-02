package update

import (
	"context"
	"fmt"

	"github.com/alex-muller/ankiai/internal/module/frequency"
	word2 "github.com/alex-muller/ankiai/internal/module/word"
)

func New(
	freq frequency.FrequencyService,
	wordsRepo *word2.Repository,
) Service {
	return Service{
		freq:      freq,
		wordsRepo: wordsRepo,
	}
}

type Service struct {
	freq      frequency.FrequencyService
	wordsRepo *word2.Repository
}

func (a Service) Run(ctx context.Context) error {
	err := a.updateFreq(ctx)
	if err != nil {
		return fmt.Errorf(`update freq: %w`, err)
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
			return fmt.Errorf(`run phrases: %w`, err)
		}
	}

	return nil
}
