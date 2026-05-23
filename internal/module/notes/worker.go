package notes

import (
	"context"

	"github.com/alex-muller/ankiai/internal/module/word"
)

func NewWorker(wordsRepo *word.Repository, cardRepo *Repo) *Worker {
	return &Worker{
		cardMaker: newMaker(cardRepo, wordsRepo),
		frequency: newFrequency(cardRepo),
	}
}

type Worker struct {
	cardMaker *maker
	frequency frequency
}

func (a *Worker) Run(ctx context.Context) {
	go a.cardMaker.run(ctx)
	go a.frequency.run(ctx)
}
