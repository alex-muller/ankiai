package run

import "github.com/alex-muller/ankiai/internal/module/telegram"

func Telegram(service *telegram.Service) {
	service.Run()
}
