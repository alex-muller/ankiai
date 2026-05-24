package run

import (
	"context"
	"log/slog"

	"github.com/alex-muller/ankiai/internal/lib/logger"
	"github.com/alex-muller/ankiai/internal/module/lexicographer"
	"github.com/alex-muller/ankiai/internal/module/notes"
)

func Daemon(
	ctx context.Context,
	lex *lexicographer.Service,
	cardsWorker *notes.Worker,
	ttsWorker *notes.Tts,
) {
	l := logger.Logger.With(slog.String("component", "daemon"))
	l.Info("daemon started")

	// go lex.Run(ctx)
	// go cardsWorker.Run(ctx)
	go ttsWorker.Run(ctx)

	<-ctx.Done()
}
