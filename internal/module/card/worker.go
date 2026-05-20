package card

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/alex-muller/ankiai/internal/lib/wp"
	"github.com/alex-muller/ankiai/internal/module/word"
)

func NewWorker(wordsRepo *word.Repository) *Worker {
	return &Worker{
		wordsRepo: wordsRepo,
	}
}

type Worker struct {
	wordsRepo *word.Repository
	log       *slog.Logger
}

func (a Worker) Run(ctx context.Context) {
	ch := make(chan word.Word)

	go func() {
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

				for _, addedWord := range addedWords {
					ch <- addedWord
				}
				time.Sleep(1 * time.Second)
			}
		}
	}()

	pool := wp.NewWorkerPool(10, 10000)
	pool.Start()

	go func() {
		var i int
		for word_ := range ch {
			i++
			taskID := i
			task := wp.Task{
				ID:      taskID,
				Payload: word_,
				Process: func(ctx context.Context, word_ any) error {
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
	}()
}

func (a Worker) processWord(ctx context.Context, word_ word.Word) error {
	time.Sleep(time.Second)
	a.log.Debug(`method`, `processWord`)
	return nil
}
