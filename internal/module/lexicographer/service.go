package lexicographer

import (
	"context"
	"log/slog"

	"github.com/alex-muller/ankiai/internal/lib/logger"
	"github.com/alex-muller/ankiai/internal/lib/wp"
	"github.com/alex-muller/ankiai/internal/module/word"
)

func New(repository *word.Repository) *Service {
	return &Service{
		repository: repository,
	}
}

type Service struct {
	repository *word.Repository
}

func (a Service) Run(ctx context.Context) {
	l := logger.Logger.With(slog.String("component", "lexicographer"))

	addedWords := a.repository.GetAdded(ctx)

	// Create worker pool: 5 workers, max 60 tasks per minute (1 task per second average)
	pool := wp.NewWorkerPool(10, 20)
	pool.Start()

	// Example: Submit some test tasks
	go func() {
		var i int
		for word_, err := range addedWords {
			if err != nil {
				slog.Error(`Failed to add word`, slog.String(`error`, err.Error()))
				break
			}
			i++
			taskID := i
			task := wp.Task{
				ID:      taskID,
				Payload: word_,
				Process: func(ctx context.Context, word any) error {
					// Example task processing logic
					l.Info("executing task", slog.Any("payload", word))
					return nil
				},
			}

			if err := pool.Submit(task); err != nil {
				l.Error("failed to submit task",
					slog.Int("task_id", taskID),
					slog.String("error", err.Error()))
			}
		}

		l.Info(`task finished`)
	}()
}
