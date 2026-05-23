package export

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/alex-muller/ankiai/internal/config"
)

func NewExporter(conf config.Config) *Exporter {
	return &Exporter{
		conf: conf,
	}
}

type Exporter struct {
	conf config.Config
}

func (a Exporter) Run(ctx context.Context) error {
	err := a.checkAndCreateDeck(ctx)
	if err != nil {
		return fmt.Errorf(`check and create deck: %w`, err)
	}

	err = a.checkAndCreateModel(ctx)
	if err != nil {
		return fmt.Errorf(`check and create model: %w`, err)
	}

	return err
}

func (a Exporter) checkAndCreateModel(ctx context.Context) error {
	ok, err := a.hasModel(ctx)
	if err != nil {
		return fmt.Errorf(`has model: %w`, err)
	}

	if ok {
		fmt.Println("model already exists")
		return nil
	}

	fmt.Println(fmt.Sprintf("model not found. creating new model [%s]", a.conf.AnkiModel))

	err = a.createModel(ctx)
	if err != nil {
		return fmt.Errorf(`create model: %w`, err)
	}

	return nil
}

func (a Exporter) checkAndCreateDeck(ctx context.Context) error {
	has, err := a.hasDeck(ctx, a.conf.AnkiDeck)
	if err != nil {
		return fmt.Errorf(`check deck: %w`, err)
	}

	if has {
		fmt.Println("deck already exists")
		return nil
	}

	fmt.Println(fmt.Sprintf("deck not found. creating new deck [%s]", a.conf.AnkiDeck))

	err = a.createDeck(ctx, a.conf.AnkiDeck)
	if err != nil {
		return fmt.Errorf(`create deck: %w`, err)
	}

	return nil
}

func (a Exporter) hasModel(ctx context.Context) (bool, error) {
	ankiRequest_ := ankiRequest{
		Action:  "modelNames",
		Version: 6,
	}

	resp, err := a.makeRequest(ctx, ankiRequest_)
	if err != nil {
		return false, fmt.Errorf(`make request: %w`, err)
	}

	results := make([]string, 0)
	err = json.Unmarshal(resp.Result, &results)
	if err != nil {
		return false, fmt.Errorf(`unmarshal deck: %w`, err)
	}

	for _, result := range results {
		if result == a.conf.AnkiModel {
			return true, nil
		}
	}

	return false, nil
}

func (a Exporter) createModel(ctx context.Context) error {
	// CSS выносим в отдельную константу для читаемости
	const css = `
.card { font-family: Arial; font-size: 18px; text-align: center; color: #333; background-color: #fdfdfd; padding: 20px; }
.sentence-box { font-size: 22px; margin: 20px 0; }
.translation-box { color: #555; font-style: italic; margin-top: 10px; }
.grammar-hint { font-size: 14px; color: #777; background: #eee; padding: 5px; display: inline-block; border-radius: 4px; }
.frequency-badge { font-size: 12px; color: #999; text-align: right; }
.listening-hint { font-size: 14px; color: #aaa; margin-top: 50px; }
.header-badges { display: flex; justify-content: space-between; }
.pos-badge { background-color: #e0e0e0; color: #555; padding: 2px 6px; border-radius: 4px; font-weight:bold; font-size: 12px; }
.extra-info { margin-top: 15px; font-size: 14px; text-align: left; background: #fafafa; padding: 10px; border-left: 3px solid #ccc; }
.word-translation { font-size: 18px; color: #d32f2f; }
.cloze { font-weight: bold; color: #0066cc; }

/* МАГИЯ ДЛЯ РЕЖИМА АУДИРОВАНИЯ (ПОДСВЕТКА МАРКЕРОМ) */
.listening-reveal-mode .cloze { color: #000 !important; background-color: #ffeb3b !important; font-weight: bold; border-radius: 3px; padding: 0 4px; }
`

	const frontHTML = `
{{#c1}}
<div class="header-badges">
	{{#part_of_speech}}<span class="pos-badge">{{part_of_speech}}</span>{{/part_of_speech}}
	<span class="frequency-badge">Freq: {{frequency}}</span>
</div>
<h3>{{definition_en}}</h3>
{{#definition_ru}}<div style="font-size: 14px; color: gray;">{{definition_ru}}</div>{{/definition_ru}}
<br>
<div class="sentence-box">{{cloze:marked_sentence}}</div>
{{/c1}}

{{#c2}}
<div style="margin-top: 50px;">{{audio}}</div>
<div class="listening-hint">(Слушай предложение и постарайся понять контекст)</div>
<div style="display:none;">{{cloze:marked_sentence}}</div>
{{/c2}}
`

	const backHTML = `
{{#c1}}
<div class="header-badges">
	{{#part_of_speech}}<span class="pos-badge">{{part_of_speech}}</span>{{/part_of_speech}}
	<span class="frequency-badge">Freq: {{frequency}}</span>
</div>
<h3>{{definition_en}}</h3>
<br>
<div class="sentence-box">{{cloze:marked_sentence}}</div>
<hr id=answer>
<div class="translation-box">
	<div class="word-translation"><b>{{lemma}}</b> — {{translation_ru}}</div>
	<div style="margin-top: 8px;">{{translation}}</div>
</div>
<div class="extra-info">
	{{#synonyms}}<div><b>Synonyms:</b> {{synonyms}}</div>{{/synonyms}}
	{{#grammar_note}}<div style="margin-top:5px;">💡 {{grammar_note}}</div>{{/grammar_note}}
</div>
<div style="margin-top: 20px;">{{audio}}</div>
{{/c1}}

{{#c2}}
<div class="listening-reveal-mode">
	<div class="sentence-box">{{cloze:marked_sentence}}</div>
</div>
<hr id=answer>
<h3>{{definition_en}}</h3>
<div class="translation-box">
	<div class="word-translation"><b>{{lemma}}</b> — {{translation_ru}}</div>
	<div style="margin-top: 8px;">{{translation}}</div>
</div>
{{/c2}}
`

	ankiRequest_ := ankiRequest{
		Action:  "createModel",
		Version: 6,
		Params: CreateModelParams{
			ModelName: a.conf.AnkiModel,
			InOrderFields: []string{
				"id",
				"lemma",
				"target_word_form",
				"marked_sentence",
				"translation",
				"grammar_note",
				"synonyms",
				"part_of_speech",
				"definition_en",
				"definition_ru",
				"translation_ru",
				"audio",
				"frequency",
			},
			Css:     css,
			IsCloze: true,
			CardTemplates: []CreateModelParamsTemplate{
				{
					// У Cloze модели должен быть строго ОДИН шаблон в массиве
					Name:  "AnkiAI Dual Cloze",
					Front: frontHTML,
					Back:  backHTML,
				},
			},
		},
	}

	resp, err := a.makeRequest(ctx, ankiRequest_)
	_ = resp
	if err != nil {
		return fmt.Errorf(`make request: %w`, err)
	}

	return nil
}

func (a Exporter) hasDeck(ctx context.Context, name string) (bool, error) {
	ankiRequest_ := ankiRequest{
		Action:  "deckNamesAndIds",
		Version: 6,
	}

	resp, err := a.makeRequest(ctx, ankiRequest_)
	if err != nil {
		return false, fmt.Errorf(`make request: %w`, err)
	}

	results := make(map[string]int)
	err = json.Unmarshal(resp.Result, new(results))
	if err != nil {
		return false, fmt.Errorf(`unmarshal deck: %w`, err)
	}

	_, ok := results[name]

	return ok, nil
}

func (a Exporter) createDeck(ctx context.Context, name string) error {
	ankiRequest_ := ankiRequest{
		Action:  "createDeck",
		Version: 6,
		Params: map[string]string{
			"deck": name,
		},
	}

	_, err := a.makeRequest(ctx, ankiRequest_)
	if err != nil {
		return fmt.Errorf(`make request: %w`, err)
	}

	return nil
}

func (a Exporter) makeRequest(ctx context.Context, request ankiRequest) (ankiResponse, error) {
	var resp ankiResponse
	payload, err := json.Marshal(request)
	if err != nil {
		return resp, fmt.Errorf(`json marshal: %w`, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.conf.AnkiConnectUrl, bytes.NewBuffer(payload)) // TODO url to env
	if err != nil {
		return resp, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp_, err := http.DefaultClient.Do(req)
	if err != nil {
		return resp, fmt.Errorf("do request: %w", err)
	}
	defer resp_.Body.Close()

	body, err := io.ReadAll(resp_.Body)

	if err := json.Unmarshal(body, &resp); err != nil {
		return resp, fmt.Errorf("unmarshal response: %w", err)
	}

	return resp, nil
}

type ankiRequest struct {
	Action  string `json:"action"`
	Version int    `json:"version"`
	Params  any    `json:"params,omitempty"`
}

type ankiResponse struct {
	Result json.RawMessage `json:"result"`
	Error  string          `json:"error"`
}

type CreateModelParams struct {
	ModelName     string                      `json:"modelName"`
	InOrderFields []string                    `json:"inOrderFields"`
	Css           string                      `json:"css"`
	IsCloze       bool                        `json:"isCloze"`
	CardTemplates []CreateModelParamsTemplate `json:"cardTemplates"`
}

type CreateModelParamsTemplate struct {
	Name  string `json:"Name"`
	Front string `json:"Front"`
	Back  string `json:"Back"`
}
