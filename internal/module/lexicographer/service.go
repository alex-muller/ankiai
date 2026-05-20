package lexicographer

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/alex-muller/ankiai/internal/lib/logger"
	"github.com/alex-muller/ankiai/internal/lib/wp"
	"github.com/alex-muller/ankiai/internal/module/word"
)

func New(repository *word.Repository, apiKey string) *Service {
	return &Service{
		repository: repository,
		log:        logger.Logger.With(slog.String("component", "lexicographer")),
		apiKey:     apiKey,
	}
}

type Service struct {
	repository *word.Repository
	log        *slog.Logger
	apiKey     string
}

func (a Service) Run(ctx context.Context) {
	ch := make(chan word.Word)

	go func() {
		for {
			select {
			case <-ctx.Done():
				close(ch)
				return
			default:
				addedWords, err := a.repository.GetAdded(ctx)
				if err != nil {
					a.log.Error(`get words to process`, slog.String("error", err.Error()))
					close(ch)
				}

				for _, addedWord := range addedWords {
					ch <- addedWord
				}
				time.Sleep(1 * time.Second)
			}
		}
	}()

	// Create worker pool: 5 workers, max 60 tasks per minute (1 task per second average)
	pool := wp.NewWorkerPool(10, 100)
	pool.Start()

	// Example: Submit some test tasks
	go func() {
		var i int
		for word_ := range ch {
			i++
			taskID := i
			task := wp.Task{
				ID:      taskID,
				Payload: word_,
				Process: func(ctx context.Context, word_ any) error {
					w, ok := word_.(word.Word)
					if !ok {
						return errors.New(`invalid word type`)
					}
					return a.processWord(ctx, w)
				},
			}

			if err := pool.Submit(task); err != nil {
				a.log.Error("failed to submit task",
					slog.Int("task_id", taskID),
					slog.String("error", err.Error()))
			}
		}

		a.log.Debug(`task finished`)
	}()
}

func (a Service) processWord(ctx context.Context, word_ word.Word) error {
	l := a.log.With(`method`, `processWord`)

	// Parse prompt template
	var requestData GeminiRequest
	if err := json.Unmarshal([]byte(prompt(word_.Word)), &requestData); err != nil {
		l.Error("failed to parse prompt template", slog.String("error", err.Error()))
		return err
	}

	// Inject word into request
	requestData.Contents[0].Parts[0].Text = "Проанализируй выражение: '" + word_.Word + "'"

	// Marshal request to JSON
	jsonData, err := json.Marshal(requestData)
	if err != nil {
		l.Error("failed to marshal request", slog.String("error", err.Error()))
		return err
	}

	// Create HTTP request
	url := "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash:generateContent?key=" + a.apiKey

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		l.Error("failed to create HTTP request", slog.String("error", err.Error()))
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		l.Error("failed to send HTTP request", slog.String("error", err.Error()))
		return err
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		l.Error("failed to read response body", slog.String("error", err.Error()))
		return err
	}

	// Check status code
	if resp.StatusCode == http.StatusOK {
		word_.RawJSON = string(body)
	} else {
		word_.ErrorLog = string(body)
	}

	word_.Status = word.StatusProcessed

	err = a.repository.Update(ctx, word_)
	if err != nil {
		return fmt.Errorf(`update: %w`, err)
	}

	return nil
}

type GeminiRequest struct {
	SystemInstruction struct {
		Parts []struct {
			Text string `json:"text"`
		} `json:"parts"`
	} `json:"systemInstruction"`
	Contents []struct {
		Parts []struct {
			Text string `json:"text"`
		} `json:"parts"`
	} `json:"contents"`
	GenerationConfig map[string]interface{} `json:"generationConfig"`
}

type GeminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

func prompt(word string) string {
	return `
{
  "systemInstruction": {
    "parts": [
      {
        "text": "Ты — профессиональный лексикограф и эксперт по иммерсивному изучению языков. Твоя задача — проанализировать английское слово или фразу и вернуть структурированные данные. Правила:\n1. Полисемия и Часть речи: Покрой основные смыслы и обязательно укажи часть речи (part_of_speech) для каждого.\n2. Перевод: Для каждого смысла дай точный перевод самого слова или фразы на русский язык (translation_ru).\n3. Примеры и Морфологическое разнообразие: Сгенерируй столько примеров, сколько нужно, чтобы покрыть морфологическое разнообразие слова. КРИТИЧЕСКИ ВАЖНО: целевое слово должно менять свою форму от примера к примеру (глаголы меняют время/форму, существительные - число и т.д.). В 'grammar_note' укажи использованную форму.\n4. ДЕРИВАЦИЯ (Производные слова): Если целевое слово образует часто используемые производные (например, от глагола образуются прилагательные с -ed/-ing или существительные), обязательно выдели их в отдельные смыслы (senses) с указанием соответствующей части речи.\n5. ВАЖНО (РАЗМЕТКА): Целевое слово или фраза в каждом примере должны встречаться строго один раз. Обязательно выделяй двойными звездочками ИМЕННО ту морфологическую форму, которая использована в тексте. Если это фразовый глагол, выделяй обе части (например: 'He is **giving** it **up**'). Не выделяй слово в переводе.\n6. Форма слова: Для каждого примера выведи в поле 'target_word_form' ту точную форму слова/фразы, которая была использована.\n7. Перевод примеров и Синонимы: Точный перевод предложения (definition_ru) и 2-3 синонима для каждого смысла."
      }
    ]
  },
  "contents": [
    {
      "parts": [
        {
          "text": "Проанализируй выражение: 'designate'"
        }
      ]
    }
  ],
  "generationConfig": {
    "responseMimeType": "application/json",
    "responseSchema": {
      "type": "OBJECT",
      "properties": {
        "lemma": { "type": "STRING" },
        "senses": {
          "type": "ARRAY",
          "items": {
            "type": "OBJECT",
            "properties": {
              "part_of_speech": { "type": "STRING", "description": "Например: noun, verb, adjective, phrasal verb" },
              "translation_ru": { "type": "STRING", "description": "Точный перевод самого слова" },
              "definition_ru": { "type": "STRING", "description": "Краткое толкование смысла на русском" },
              "synonyms": { "type": "ARRAY", "items": { "type": "STRING" } },
              "examples": {
                "type": "ARRAY",
                "items": {
                  "type": "OBJECT",
                  "properties": {
                    "grammar_note": { "type": "STRING", "description": "Указание формы/времени" },
                    "target_word_form": { "type": "STRING", "description": "Точная форма слова, использованная в примере (чистый текст без звездочек)" },
                    "marked_sentence": { "type": "STRING", "description": "Предложение с разметкой **word**" },
                    "translation": { "type": "STRING" }
                  },
                  "required": ["grammar_note", "target_word_form", "marked_sentence", "translation"]
                }
              }
            },
            "required": ["part_of_speech", "translation_ru", "definition_ru", "synonyms", "examples"]
          }
        }
      },
      "required": ["lemma", "senses"]
    }
  }
}
`
}
