package update

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/alex-muller/ankiai/internal/module/frequency"
	"github.com/alex-muller/ankiai/internal/module/lexicographer"
	"github.com/alex-muller/ankiai/internal/module/logger"
	"github.com/alex-muller/ankiai/internal/module/notes"
	"github.com/alex-muller/ankiai/internal/module/tts"
	word2 "github.com/alex-muller/ankiai/internal/module/word"
	"golang.org/x/sync/errgroup"
)

func New(
	freq frequency.FrequencyService,
	wordsRepo *word2.Repository,
	notesRepo *notes.Repo,
	lex *lexicographer.Service,
	tts *tts.Tts,
) Service {
	return Service{
		freq:      freq,
		wordsRepo: wordsRepo,
		notesRepo: notesRepo,
		lex:       lex,
		tts:       tts,
		logger:    logger.NewLogger(3),
	}
}

type Service struct {
	freq      frequency.FrequencyService
	wordsRepo *word2.Repository
	notesRepo *notes.Repo
	lex       *lexicographer.Service
	tts       *tts.Tts
	logger    *logger.MultiLineLogger
}

func (a Service) Run(ctx context.Context) error {
	group, ctx := errgroup.WithContext(ctx)

	group.Go(func() error {
		err := a.updateFreq(ctx)
		if err != nil {
			return fmt.Errorf(`update freq: %w`, err)
		}
		return nil
	})

	group.Go(func() error {
		err := a.updatePlTranslate(ctx)
		if err != nil {
			return fmt.Errorf(`update pl translate: %w`, err)
		}

		return nil
	})

	group.Go(func() error {
		err := a.updatePlTTS(ctx)
		if err != nil {
			return fmt.Errorf(`update pl translate: %w`, err)
		}

		return nil
	})

	err := group.Wait()
	if err != nil {
		return err
	}

	return nil

}

func (a Service) updatePlTTS(ctx context.Context) error {
	notes_, err := a.notesRepo.FindManyForPolishTTSUpdate(ctx, 0)
	if err != nil {
		return fmt.Errorf(`find notes: %w`, err)
	}

	total := len(notes_)
	var count int

	limitPerMinute := 10
	workers := 10

	ch := make(chan notes.Note)

	go func() {
		defer close(ch)
		for _, note := range notes_ {
			select {
			case <-ctx.Done():
				return
			case ch <- note:
			}

			time.Sleep(time.Minute / time.Duration(limitPerMinute))
		}
	}()

	wg := sync.WaitGroup{}
	mu := sync.Mutex{}

	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case n, ok := <-ch:
					if !ok {
						return
					}
					err = a.updateOnePlTtsNote(ctx, n)
					if err != nil {
						panic(err)
					}

					mu.Lock()
					count++
					mu.Unlock()
					a.logger.UpdateLine(2, fmt.Sprintf("PL TTS updated %d of %d", count, total))
				}
			}
		}()
	}

	wg.Wait()

	return nil
}

func (a Service) updateOnePlTtsNote(ctx context.Context, n notes.Note) error {
	err := a.tts.UpdatePl(ctx, n)
	if err != nil {
		return fmt.Errorf(`update pl translate: %w`, err)
	}
	return nil
}

func (a Service) updatePlTranslate(ctx context.Context) error {
	notes_, err := a.notesRepo.FindManyForPolishTranslateUpdate(ctx, 0)
	if err != nil {
		return fmt.Errorf(`find notes: %w`, err)
	}

	total := len(notes_)
	var count int

	limitPerMinute := 100
	workers := 10

	ch := make(chan notes.Note)

	go func() {
		defer close(ch)
		for _, note := range notes_ {
			select {
			case <-ctx.Done():
				return
			case ch <- note:
			}

			time.Sleep(time.Minute / time.Duration(limitPerMinute))
		}
	}()

	wg := sync.WaitGroup{}
	mu := sync.Mutex{}

	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case n, ok := <-ch:
					if !ok {
						return
					}
					err = a.updatePlOneNote(ctx, n)
					if err != nil {
						panic(err)
					}

					mu.Lock()
					count++
					mu.Unlock()
					a.logger.UpdateLine(1, fmt.Sprintf("PL translate updated %d of %d", count, total))
				}
			}
		}()
	}

	wg.Wait()

	return nil
}

func (a Service) updatePlOneNote(ctx context.Context, n notes.Note) error {
	translate, err := a.lex.GetPlTranslate(ctx, n.TargetWordForm, n.GetSentenceEn(), n.Translation, n.DefinitionEn, n.DefinitionRu)
	if err != nil {
		return fmt.Errorf(`get pl translate: %w`, err)
	}

	n.DefinitionPl = translate.DefinitionPl
	n.TranslationPl = translate.ExampleTranslationPl
	n.UpdatedAt = time.Now().UTC()

	err = a.notesRepo.Update(ctx, n)
	if err != nil {
		return fmt.Errorf(`update pl note: %w`, err)
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
		a.logger.UpdateLine(0, fmt.Sprintf("Process frequency. Left: %d", left))

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
				a.logger.UpdateLine(0, `Frequency limit exceeded. Sleep`)
				time.Sleep(time.Minute * 10)
				continue
			}
			return fmt.Errorf(`run phrases: %w`, err)
		}
	}

	return nil
}
