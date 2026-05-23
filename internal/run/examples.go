package run

import (
	"context"

	"github.com/alex-muller/ankiai/internal/module/lexicographer"
)

func Examples(ctx context.Context, ls *lexicographer.Service) {
	ls.RunOnce(ctx)
}
