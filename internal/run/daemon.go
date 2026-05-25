package run

import (
	"context"
	"log/slog"

	"github.com/alex-muller/ankiai/internal/lib/logger"
	"github.com/alex-muller/ankiai/internal/module/frequency"
	"github.com/alex-muller/ankiai/internal/module/lexicographer"
	"github.com/alex-muller/ankiai/internal/module/notes"
	"github.com/alex-muller/ankiai/internal/module/tts"
)

func Daemon(
	ctx context.Context,
	lex *lexicographer.Service,
	frequency frequency.FrequencyService,
	notesMaker *notes.Maker,
	ttsWorker *tts.Tts,
) {
	l := logger.Logger.With(slog.String("component", "daemon"))
	l.Info("daemon started")

	// go lex.Run(ctx)
	go notesMaker.Run(ctx)
	go frequency.Run(ctx)
	go ttsWorker.Run(ctx)

	<-ctx.Done()
}
