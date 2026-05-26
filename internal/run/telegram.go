package run

import (
	"context"

	"github.com/alex-muller/ankiai/internal/module/telegram"
)

func Telegram(ctx context.Context, service *telegram.Service) {
	service.Run(ctx)
}
