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

	requestData := promptWithPl(word_.Word)

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

func promptWithPl(word string) GeminiRequest {
	return GeminiRequest{
		SystemInstruction: &Content{
			Parts: []Part{
				{
					Text: `Ты — профессиональный лексикограф и эксперт по иммерсивному изучению языков. Твоя задача — проанализировать английское слово или фразу и вернуть структурированные данные. Правила:
1. Полисемия и Часть речи: Покрой основные смыслы и обязательно укажи часть речи (part_of_speech) на английском.
2. Перевод: Для каждого смысла дай точный перевод самого слова или фразы на русский язык (translation_ru) и на польский язык (translation_pl). Перевод на польский должен быть максимально естественным и идиоматичным для носителей языка (избегай буквального перевода английских конструкций).
3. Примеры и Морфологическое разнообразие: Сгенерируй столько примеров, сколько нужно, чтобы предотвратить морфологическое разнообразие слова. КРИТИЧЕСКИ ВАЖНО: целевое слово должно менять свою форму от примера к примеру. В 'grammar_note' кратко укажи использованную форму СТРОГО НА АНГЛИЙСКОМ ЯЗЫКЕ (например: Past Simple, Plural Noun, Gerund). Для каждого примера дай перевод предложения на русский (translation) и естественный перевод на польский (translation_pl).
4. ДЕРИВАЦИЯ (Производные слова): Если целевое слово образует часто используемые производные, обязательно выдели их в отдельные смыслы (senses) с указанием соответствующей части речи.
5. ВАЖНО (РАЗМЕТКА): Целевое слово или фраза в каждом примере должны встречаться строго один раз. Обязательно выделяй двойными звездочками ИМЕННО ту морфологическую форму, которая использована в тексте. Не выделяй слово в переводе.
6. Форма слова: Для каждого примера выведи в поле 'target_word_form' ту точную форму слова/фразы, которая была использована.
7. Определения и Синонимы: Для каждого смысла напиши простое и понятное толкование на английском (definition_en), краткий перевод на русский (definition_ru), естественное толкование смысла на польском языке (definition_pl) и 2-3 синонима на английском.`,
				},
			},
		},
		Contents: []Content{
			{
				Parts: []Part{
					{
						Text: "Проанализируй выражение: '" + word + "'",
					},
				},
			},
		},
		GenerationConfig: &GenerationConfig{
			ResponseMimeType: "application/json",
			ResponseSchema: &Schema{
				Type:     "OBJECT",
				Required: []string{"lemma", "senses"},
				Properties: map[string]*Schema{
					"lemma": {
						Type: "STRING",
					},
					"senses": {
						Type: "ARRAY",
						Items: &Schema{
							Type: "OBJECT",
							Required: []string{
								"part_of_speech",
								"translation_ru",
								"translation_pl",
								"definition_en",
								"definition_ru",
								"definition_pl",
								"synonyms",
								"examples",
							},
							Properties: map[string]*Schema{
								"part_of_speech": {
									Type:        "STRING",
									Description: "e.g., noun, verb, adjective",
								},
								"translation_ru": {
									Type:        "STRING",
									Description: "Точный перевод самого слова на русский",
								},
								"translation_pl": {
									Type:        "STRING",
									Description: "Naturalny i idiomatyczny przekład słowa na język polski",
								},
								"definition_en": {
									Type:        "STRING",
									Description: "Clear English explanation of the meaning",
								},
								"definition_ru": {
									Type:        "STRING",
									Description: "Kratkoe tolkovanie smysla na russkom",
								},
								"definition_pl": {
									Type:        "STRING",
									Description: "Krótkie i zrozumiałe wyjaśnienie znaczenia po polsku",
								},
								"synonyms": {
									Type: "ARRAY",
									Items: &Schema{
										Type: "STRING",
									},
								},
								"examples": {
									Type: "ARRAY",
									Items: &Schema{
										Type: "OBJECT",
										Required: []string{
											"grammar_note",
											"target_word_form",
											"marked_sentence",
											"translation",
											"translation_pl",
										},
										Properties: map[string]*Schema{
											"grammar_note": {
												Type:        "STRING",
												Description: "e.g., Past Simple (V2), Present Continuous, Plural",
											},
											"target_word_form": {
												Type:        "STRING",
												Description: "Точная форма слова, использованная в примере",
											},
											"marked_sentence": {
												Type:        "STRING",
												Description: "Предложение с разметкой **word**",
											},
											"translation": {
												Type:        "STRING",
												Description: "Перевод предложения на русский",
											},
											"translation_pl": {
												Type:        "STRING",
												Description: "Naturalne tłumaczenie całego zdania na język polski",
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (a *Service) GetPlTranslate(
	ctx context.Context,
	targetWordForm,
	exampleEn,
	exampleTranslationRu,
	definitionEn,
	definitionRu string,
) (PlPatchResponse, error) {
	var out PlPatchResponse

	req := promptOnlyPlExamplePatch(targetWordForm, exampleEn, exampleTranslationRu, definitionEn, definitionRu)

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

func promptOnlyPlExamplePatch(targetWordForm, exampleEn, exampleTranslationRu, definitionEn, definitionRu string) GeminiRequest {
	return GeminiRequest{
		SystemInstruction: &Content{
			Parts: []Part{
				{
					Text: "Ты — эксперт-переводчик со специализацией на паре русский-польский и английский-польский. " +
						"Твоя задача — проанализировать английский пример и его русский контекст, а затем вернуть перевод " +
						"и толкование на естественном, живом и идиоматичном польском языке.\n\n" +
						"ПРАВИЛА ПЕРЕВОДА:\n" +
						"1. Используй предоставленный русский перевод предложения (example_translation_ru) и русское " +
						"толкование (definition_ru) как точный ориентир смысла. Польский перевод должен строго " +
						"соответствовать этому контексту.\n" +
						"2. Переведи английское определение (definition_en) на польский язык (definition_pl).\n" +
						"3. Переведи само английское предложение-пример (example_en) на польский язык (example_translation_pl). " +
						"Перевод предложения должен звучать максимально естественно для носителей языка, " +
						"избегай кальки английских или русских фраз.",
				},
			},
		},
		Contents: []Content{
			{
				Parts: []Part{
					{
						Text: `Целевое слово (форма): '` + targetWordForm + `'\n\nОРИЕНТИР СМЫСЛА:\nТолкование на русском: '` +
							definitionRu + `'\nПеревод предложения на русский: '` + exampleTranslationRu +
							`'\n\nАНГЛИЙСКИЙ КОНТЕКСТ:\nОпределение на английском: '` + definitionEn +
							`'\nПредложение на английском: '` + exampleEn + `'`,
					},
				},
			},
		},
		GenerationConfig: &GenerationConfig{
			ResponseMimeType: "application/json",
			ResponseSchema: &Schema{
				Type:        "OBJECT",
				Description: "",
				Properties: map[string]*Schema{
					"definition_pl": {
						Type:        "STRING",
						Description: "Wyjaśnienie znaczenia słowa po polsku, odpowiadające kontekstowi",
					},
					"example_translation_pl": {
						Type:        "STRING",
						Description: "Naturalne i idiomatyczne tłumaczenie całego zdania przykładowego na język polski",
					},
				},
				Required: []string{"definition_pl", "example_translation_pl"},
			},
		},
	}
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

// RequestPayload — корневая структура для обоих JSON.
type GeminiRequest struct {
	SystemInstruction *Content          `json:"systemInstruction,omitempty"`
	Contents          []Content         `json:"contents"`
	GenerationConfig  *GenerationConfig `json:"generationConfig,omitempty"`
}

type Content struct {
	Parts []Part `json:"parts"`
}

type Part struct {
	Text string `json:"text"`
}

type GenerationConfig struct {
	ResponseMimeType string  `json:"responseMimeType,omitempty"`
	ResponseSchema   *Schema `json:"responseSchema,omitempty"`
}

// Schema — рекурсивная структура, описывающая JSON Schema.
type Schema struct {
	Type        string `json:"type"`                  // Например: "OBJECT", "ARRAY", "STRING"
	Description string `json:"description,omitempty"` // Описание поля, если есть

	// Properties использует map, так как ключи (имена полей) неизвестны заранее
	// и отличаются в разных запросах (например, "definition_pl" или "senses").
	Properties map[string]*Schema `json:"properties,omitempty"`

	// Items содержит описание элементов массива, если Type == "ARRAY".
	// Используется указатель для рекурсии.
	Items *Schema `json:"items,omitempty"`

	Required []string `json:"required,omitempty"` // Список обязательных полей
}
