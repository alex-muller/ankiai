package run

import (
	"context"
	"log/slog"

	"github.com/alex-muller/ankiai/internal/lib/logger"
	"github.com/alex-muller/ankiai/internal/module/lexicographer"
)

func Daemon(ctx context.Context, lex *lexicographer.Service) {
	l := logger.Logger.With(slog.String("component", "daemon"))
	l.Info("daemon started")

	go lex.Run(ctx)

	<-ctx.Done()
}
