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
	"sync"
	"sync/atomic"
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
	counter    atomic.Int64
}

func (a *Service) RunDaemon(ctx context.Context) {
	a.run(ctx, true)
}

func (a *Service) RunOnce(ctx context.Context) {
	a.run(ctx, false)
}

func (a *Service) run(ctx context.Context, asDaemon bool) {
	ch := make(chan word.Word)
	wg := &sync.WaitGroup{}
	wg.Add(1)

	go func() {
		defer close(ch)
		wg.Done()
		for {
			select {
			case <-ctx.Done():
				close(ch)
				return
			default:
				a.log.Info(`try to get words`)
				addedWords, err := a.repository.GetByStatus(ctx, word.StatusNew)
				if err != nil {
					a.log.Error(`get words to process`, slog.String("error", err.Error()))
					close(ch)
					return
				}

				if len(addedWords) == 0 {
					if asDaemon {
						time.Sleep(1 * time.Minute)
						continue
					}
					return
				}

				for _, addedWord := range addedWords {
					wg.Add(1)
					ch <- addedWord
				}

				wg.Wait()
			}
		}
	}()

	// Create worker pool: 5 workers, max 60 tasks per minute (1 task per second average)
	pool := wp.NewWorkerPool(8, 500)
	pool.Start()

	var i int
	for word_ := range ch {
		i++
		taskID := i
		task := wp.Task{
			ID:      taskID,
			Payload: word_,
			Process: func(ctx context.Context, word_ any) error {
				defer wg.Done()
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
}

func (a *Service) processWord(ctx context.Context, word_ word.Word) error {
	l := a.log.With(`method`, `processWord`)
	l.Info(`start process word ` + word_.Word)

	// Parse prompt template
	var requestData GeminiRequest
	if err := json.Unmarshal([]byte(promptWithPl(word_.Word)), &requestData); err != nil {
		l.Error("failed to parse prompt template", slog.String("error", err.Error()))
		return err
	}

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

	word_.Status = word.StatusRaw

	err = a.repository.Update(ctx, word_)
	if err != nil {
		return fmt.Errorf(`update: %w`, err)
	}

	a.counter.Add(1)

	fmt.Println(fmt.Sprintf("[example generator] processed word: %s, total processed: %d", word_.Word, a.counter.Load()))

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

func promptOld(word string) string {
	return `
{
  "systemInstruction": {
    "parts": [
      {
        "text": "Ты — профессиональный лексикограф и эксперт по иммерсивному изучению языков. Твоя задача — проанализировать английское слово или фразу и вернуть структурированные данные. Правила:\n1. Полисемия и Часть речи: Покрой основные смыслы и обязательно укажи часть речи (part_of_speech) на английском.\n2. Перевод: Для каждого смысла дай точный перевод самого слова или фразы на русский язык (translation_ru).\n3. Примеры и Морфологическое разнообразие: Сгенерируй столько примеров, сколько нужно, чтобы покрыть морфологическое разнообразие слова. КРИТИЧЕСКИ ВАЖНО: целевое слово должно менять свою форму от примера к примеру. В 'grammar_note' кратко укажи использованную форму СТРОГО НА АНГЛИЙСКОМ ЯЗЫКЕ (например: Past Simple, Plural Noun, Gerund).\n4. ДЕРИВАЦИЯ (Производные слова): Если целевое слово образует часто используемые производные, обязательно выдели их в отдельные смыслы (senses) с указанием соответствующей части речи.\n5. ВАЖНО (РАЗМЕТКА): Целевое слово или фраза в каждом примере должны встречаться строго один раз. Обязательно выделяй двойными звездочками ИМЕННО ту морфологическую форму, которая использована в тексте. Не выделяй слово в переводе.\n6. Форма слова: Для каждого примера выведи в поле 'target_word_form' ту точную форму слова/фразы, которая была использована.\n7. Определения и Синонимы: Для каждого смысла напиши простое и понятное толкование на английском (definition_en), краткий перевод на русский (definition_ru) и 2-3 синонима на английском."
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
              "part_of_speech": { "type": "STRING", "description": "e.g., noun, verb, adjective" },
              "translation_ru": { "type": "STRING", "description": "Точный перевод самого слова" },
              "definition_en": { "type": "STRING", "description": "Clear English explanation of the meaning" },
              "definition_ru": { "type": "STRING", "description": "Краткое толкование смысла на русском" },
              "synonyms": { "type": "ARRAY", "items": { "type": "STRING" } },
              "examples": {
                "type": "ARRAY",
                "items": {
                  "type": "OBJECT",
                  "properties": {
                    "grammar_note": { "type": "STRING", "description": "e.g., Past Simple (V2), Present Continuous, Plural" },
                    "target_word_form": { "type": "STRING", "description": "Точная форма слова, использованная в примере" },
                    "marked_sentence": { "type": "STRING", "description": "Предложение с разметкой **word**" },
                    "translation": { "type": "STRING", "description": "Перевод предложения на русский" }
                  },
                  "required": ["grammar_note", "target_word_form", "marked_sentence", "translation"]
                }
              }
            },
            "required": ["part_of_speech", "translation_ru", "definition_en", "definition_ru", "synonyms", "examples"]
          }
        }
      },
      "required": ["lemma", "senses"]
    }
  }
}`
}

func promptWithPl(word string) string {
	return `
{
  "systemInstruction": {
    "parts": [
      {
        "text": "Ты — профессиональный лексикограф и эксперт по иммерсивному изучению языков. Твоя задача — проанализировать английское слово или фразу и вернуть структурированные данные. Правила:\n1. Полисемия и Часть речи: Покрой основные смыслы и обязательно укажи часть речи (part_of_speech) на английском.\n2. Перевод: Для каждого смысла дай точный перевод самого слова или фразы на русский язык (translation_ru) и на польский язык (translation_pl). Перевод на польский должен быть максимально естественным и идиоматичным для носителей языка (избегай буквального перевода английских конструкций).\n3. Примеры и Морфологическое разнообразие: Сгенерируй столько примеров, сколько нужно, чтобы покрыть морфологическое разнообразие слова. КРИТИЧЕСКИ ВАЖНО: целевое слово должно менять свою форму от примера к примеру. В 'grammar_note' кратко укажи использованную форму СТРОГО НА АНГЛИЙСКОМ ЯЗЫКЕ (например: Past Simple, Plural Noun, Gerund). Для каждого примера дай перевод предложения на русский (translation) и естественный перевод на польский (translation_pl).\n4. ДЕРИВАЦИЯ (Производные слова): Если целевое слово образует часто используемые производные, обязательно выдели их в отдельные смыслы (senses) с указанием соответствующей части речи.\n5. ВАЖНО (РАЗМЕТКА): Целевое слово или фраза в каждом примере должны встречаться строго один раз. Обязательно выделяй двойными звездочками ИМЕННО ту морфологическую форму, которая использована в тексте. Не выделяй слово в переводе.\n6. Форма слова: Для каждого примера выведи в поле 'target_word_form' ту точную форму слова/фразы, которая была использована.\n7. Определения и Синонимы: Для каждого смысла напиши простое и понятное толкование на английском (definition_en), краткий перевод на русский (definition_ru), естественное толкование смысла на польском языке (definition_pl) и 2-3 синонима на английском."
      }
    ]
  },
  "contents": [
    {
      "parts": [
        {
          "text": "Проанализируй выражение: '` + word + `'"
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
              "part_of_speech": { "type": "STRING", "description": "e.g., noun, verb, adjective" },
              "translation_ru": { "type": "STRING", "description": "Точный перевод самого слова на русский" },
              "translation_pl": { "type": "STRING", "description": "Naturalny i idiomatyczny przekład słowa na język polski" },
              "definition_en": { "type": "STRING", "description": "Clear English explanation of the meaning" },
              "definition_ru": { "type": "STRING", "description": "Kratkoe tolkovanie smysla na russkom" },
              "definition_pl": { "type": "STRING", "description": "Krótkie i zrozumiałe wyjaśnienie znaczenia po polsku" },
              "synonyms": { "type": "ARRAY", "items": { "type": "STRING" } },
              "examples": {
                "type": "ARRAY",
                "items": {
                  "type": "OBJECT",
                  "properties": {
                    "grammar_note": { "type": "STRING", "description": "e.g., Past Simple (V2), Present Continuous, Plural" },
                    "target_word_form": { "type": "STRING", "description": "Точная форма слова, использованная в примере" },
                    "marked_sentence": { "type": "STRING", "description": "Предложение с разметкой **word**" },
                    "translation": { "type": "STRING", "description": "Перевод предложения на русский" },
                    "translation_pl": { "type": "STRING", "description": "Naturalne tłumaczenie całego zdania na język polski" }
                  },
                  "required": ["grammar_note", "target_word_form", "marked_sentence", "translation", "translation_pl"]
                }
              }
            },
            "required": ["part_of_speech", "translation_ru", "translation_pl", "definition_en", "definition_ru", "definition_pl", "synonyms", "examples"]
          }
        }
      },
      "required": ["lemma", "senses"]
    }
  }
}`
}
