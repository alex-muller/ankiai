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
				return
			default:
				a.log.Info(`try to get words`)
				addedWords, err := a.repository.GetForExamples(ctx, word.StatusAddedFrequency)
				if err != nil {
					a.log.Error(`get words to process`, slog.String("error", err.Error()))
					return
				}

				if len(addedWords) == 0 {
					if asDaemon {
						time.Sleep(1 * time.Second)
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

	var requestData GeminiRequest

	if err := json.Unmarshal([]byte(promptWithPl(word_.Word)), &requestData); err != nil {
		l.Error("failed to parse prompt template", slog.String("error", err.Error()))
		return err
	}

	resp, err := a.makeRequest(ctx, requestData)
	if err != nil {
		l.Error("failed to make request", slog.String("error", err.Error()))
	}
	word_.RawJSON = resp

	word_.Status = word.StatusRaw

	err = a.repository.Update(ctx, word_)
	if err != nil {
		return fmt.Errorf(`update: %w`, err)
	}

	a.counter.Add(1)

	fmt.Println(fmt.Sprintf("[example generator] processed word: %s, total processed: %d", word_.Word, a.counter.Load()))

	return nil
}

func (a *Service) makeRequest(ctx context.Context, requestData GeminiRequest) (string, error) {
	// Marshal request to JSON
	jsonData, err := json.Marshal(requestData)
	if err != nil {
		return ``, fmt.Errorf(`json marshal: %w`, err)
	}

	// Create HTTP request
	url := "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash:generateContent?key=" + a.apiKey // TODO модель ИИ в конфиг

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return ``, fmt.Errorf(`new request: %w`, err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return ``, fmt.Errorf(`client do: %w`, err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ``, fmt.Errorf(`read body: %w`, err)
	}

	return string(body), nil
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

func (a *Service) GetPlTranslate(
	ctx context.Context,
	targetWordForm,
	exampleEn,
	exampleTranslationRu,
	definitionEn,
	definitionRu string,
) (PlPatchResponse, error) {
	var req GeminiRequest
	var out PlPatchResponse

	err := json.Unmarshal([]byte(promptOnlyPlExamplePatch(targetWordForm, exampleEn, exampleTranslationRu, definitionEn, definitionRu)), &req)
	if err != nil {
		return out, fmt.Errorf("unmarshal pl example: %w", err)
	}

	resp, err := a.makeRequest(ctx, req)

	var respStruct PlResponse

	err = json.Unmarshal([]byte(resp), &respStruct)
	if err != nil {
		return out, fmt.Errorf("unmarshal response: %w", err)
	}

	err = json.Unmarshal([]byte(respStruct.Candidates[0].Content.Parts[0].Text), &out)
	if err != nil {
		return out, fmt.Errorf("unmarshal text: %w", err)
	}

	return out, nil
}

func promptOnlyPlExamplePatch(targetWordForm, exampleEn, exampleTranslationRu, definitionEn, definitionRu string) string {
	return `
{
  "systemInstruction": {
    "parts": [
      {
        "text": "Ты — эксперт-переводчик со специализацией на паре русский-польский и английский-польский. Твоя задача — проанализировать английский пример и его русский контекст, а затем вернуть перевод и толкование на естественном, живом и идиоматичном польском языке.\n\nПРАВИЛА ПЕРЕВОДА:\n1. Используй предоставленный русский перевод предложения (example_translation_ru) и русское толкование (definition_ru) как точный ориентир смысла. Польский перевод должен строго соответствовать этому контексту.\n2. Переведи английское определение (definition_en) на польский язык (definition_pl).\n3. Переведи само английское предложение-пример (example_en) на польский язык (example_translation_pl). Перевод предложения должен звучать максимально естественно для носителей языка, избегай кальки английских или русских фраз."
      }
    ]
  },
  "contents": [
    {
      "parts": [
        {
          "text": "Целевое слово (форма): '` + targetWordForm + `'\n\nОРИЕНТИР СМЫСЛА:\nТолкование на русском: '` + definitionRu + `'\nПеревод предложения на русский: '` + exampleTranslationRu + `'\n\nАНГЛИЙСКИЙ КОНТЕКСТ:\nОпределение на английском: '` + definitionEn + `'\nПредложение на английском: '` + exampleEn + `'"
        }
      ]
    }
  ],
  "generationConfig": {
    "responseMimeType": "application/json",
    "responseSchema": {
      "type": "OBJECT",
      "properties": {
        "definition_pl": { "type": "STRING", "description": "Wyjaśnienie znaczenia słowa po polsku, odpowiadające kontekstowi" },
        "example_translation_pl": { "type": "STRING", "description": "Naturalne i idiomatyczne tłumaczenie całego zdania przykładowego na język polski" }
      },
      "required": ["definition_pl", "example_translation_pl"]
    }
  }
}`
}

type PlPatchResponse struct {
	DefinitionPl         string `json:"definition_pl"`
	ExampleTranslationPl string `json:"example_translation_pl"`
}

type PlResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
			Role string `json:"role"`
		} `json:"content"`
		FinishReason string `json:"finishReason"`
		Index        int    `json:"index"`
	} `json:"candidates"`
	UsageMetadata struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
		TotalTokenCount      int `json:"totalTokenCount"`
		PromptTokensDetails  []struct {
			Modality   string `json:"modality"`
			TokenCount int    `json:"tokenCount"`
		} `json:"promptTokensDetails"`
		ThoughtsTokenCount int    `json:"thoughtsTokenCount"`
		ServiceTier        string `json:"serviceTier"`
	} `json:"usageMetadata"`
	ModelVersion string `json:"modelVersion"`
	ResponseId   string `json:"responseId"`
}
