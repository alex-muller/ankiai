package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/alex-muller/ankiai/internal/config"
	"github.com/alex-muller/ankiai/internal/module/importer"
)

func New(conf config.Config, importService *importer.Service) *Service {
	return &Service{
		conf:          conf,
		importService: importService,
	}
}

type Service struct {
	conf          config.Config
	importService *importer.Service
}

func (a Service) Run(ctx context.Context) {

	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s", a.conf.TelegramBotToken)

	// Ваш Telegram ID для защиты от чужих сообщений
	var myUserID int64 = 458726892 // TODO move to env

	// offset нужен, чтобы сообщать Телеграму, какие апдейты мы уже прочитали
	offset := 0
	client := &http.Client{Timeout: 15 * time.Second}

	log.Println("Бот запущен на чистом Go и ждет слов...")

	for {
		// 1. ПОЛУЧАЕМ АПДЕЙТЫ (Long Polling)
		// timeout=10 заставляет Телеграм держать соединение 10 секунд, если нет новых сообщений
		url := fmt.Sprintf("%s/getUpdates?offset=%d&timeout=10", apiURL, offset)

		resp, err := client.Get(url)
		if err != nil {
			log.Printf("Ошибка запроса getUpdates: %v", err)
			time.Sleep(2 * time.Second) // Пауза при ошибке сети
			continue
		}

		var updates GetUpdatesResponse
		if err := json.NewDecoder(resp.Body).Decode(&updates); err != nil {
			log.Printf("Ошибка декодирования JSON: %v", err)
			resp.Body.Close()
			continue
		}
		resp.Body.Close()

		// 2. ОБРАБАТЫВАЕМ КАЖДОЕ СООБЩЕНИЕ
		for _, update := range updates.Result {
			// Сдвигаем offset, чтобы больше не получать это сообщение
			offset = update.UpdateID + 1

			// Пропускаем все, что не является текстовым сообщением (картинки, системные ивенты)
			if update.Message == nil || update.Message.Text == "" {
				continue
			}

			chatID := update.Message.Chat.ID

			// СЕКЬЮРИТИ: игнорируем чужие сообщения
			if chatID != myUserID {
				log.Printf("Заблокирован доступ от ID: %d", chatID)
				continue
			}

			words := strings.TrimSpace(update.Message.Text)

			imported, err := a.importService.ImportText(ctx, update.Message.Text)
			var replyText string
			if err != nil {
				replyText = fmt.Sprintf("❌ Ошибка: %s", err.Error())
			} else {
				if len(imported) > 0 {
					replyText = fmt.Sprintf("✅ Импортировано [%s] добавлено в очередь", strings.Join(imported, ", "))
				} else {
					replyText = `Похоже данные слова уже импортированы`
				}
			}

			// 3. ОТПРАВЛЯЕМ ОТВЕТ
			sendReq := SendMessageRequest{
				ChatID:    chatID,
				Text:      replyText,
				ParseMode: "Markdown",
			}

			reqBody, _ := json.Marshal(sendReq)
			_, err = client.Post(apiURL+"/sendMessage", "application/json", bytes.NewBuffer(reqBody))
			if err != nil {
				log.Printf("Ошибка отправки ответа: %v", err)
			} else {
				log.Printf("Обработано слово: %s", words)
			}
		}
	}
}

type GetUpdatesResponse struct {
	Ok     bool     `json:"ok"`
	Result []Update `json:"result"`
}

type Update struct {
	UpdateID int      `json:"update_id"`
	Message  *Message `json:"message"`
}

type Message struct {
	Chat Chat   `json:"chat"`
	Text string `json:"text"`
}

type Chat struct {
	ID int64 `json:"id"`
}

// --- Структура для отправки сообщений ---

type SendMessageRequest struct {
	ChatID    int64  `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode,omitempty"`
}
