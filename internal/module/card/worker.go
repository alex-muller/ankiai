package card

import (
	"context"

	"github.com/alex-muller/ankiai/internal/module/word"
)

func NewWorker(wordsRepo *word.Repository, cardRepo *CardRepo) *Worker {
	return &Worker{
		cardMaker: newMaker(cardRepo, wordsRepo),
	}
}

type Worker struct {
	cardMaker *maker
}

func (a *Worker) Run(ctx context.Context) {
	go a.cardMaker.run(ctx)
}
