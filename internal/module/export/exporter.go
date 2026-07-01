package export

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/alex-muller/ankiai/internal/config"
	"github.com/alex-muller/ankiai/internal/module/notes"
	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"
)

func NewExporter(conf config.Config, notesRepo *notes.Repo) *Exporter {
	return &Exporter{
		conf:       conf,
		clozeRegex: regexp.MustCompile(`\*\*(.*?)\*\*`),
		notesRepo:  notesRepo,
	}
}

type Exporter struct {
	conf       config.Config
	notesRepo  *notes.Repo
	clozeRegex *regexp.Regexp
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

	err = a.runExportCards(ctx)
	if err != nil {
		return fmt.Errorf(`run export cards: %w`, err)
	}

	return nil
}

func (a Exporter) Update(ctx context.Context) error {
	p := mpb.New(mpb.WithWidth(100))

	total := 0
	name := "Single Bar:"
	// create a single bar, which will inherit container's width
	bar := p.AddBar(int64(total),
		// BarFillerBuilder with custom style
		mpb.PrependDecorators(
			// display our name with one space on the right
			decor.Name(name, decor.WC{C: decor.DindentRight | decor.DextraSpace}),
			// replace ETA decorator with "done" message, OnComplete event
			decor.OnComplete(decor.AverageETA(decor.ET_STYLE_GO), "done"),
		),
		mpb.AppendDecorators(decor.Percentage()),
	)

	bar2 := p.AddBar(int64(total),
		// BarFillerBuilder with custom style
		mpb.PrependDecorators(
			// display our name with one space on the right
			decor.Name(name, decor.WC{C: decor.DindentRight | decor.DextraSpace}),
			// replace ETA decorator with "done" message, OnComplete event
			decor.OnComplete(decor.AverageETA(decor.ET_STYLE_GO), "done"),
		),
		mpb.AppendDecorators(decor.Percentage()),
	)

	bar.SetTotal(100, false)
	// simulating some work
	max := 100 * time.Millisecond
	for range 100 {
		time.Sleep(time.Duration(rand.Intn(10)+1) * max / 10)
		bar.Increment()
		bar2.Increment()
	}

	bar.Abort(true)
	bar.SetTotal(50, false)
	bar.Increment()

	// wait for our bar to complete and flush
	p.Wait()

	return nil
}

func (a Exporter) runExportCards(ctx context.Context) error {
	notes, err := a.notesRepo.GetManyByStatus(ctx, notes.ExportPending, 0)
	if err != nil {
		return fmt.Errorf(`get notes: %w`, err)
	}

	for _, note := range notes {
		err = a.exportOneNote(ctx, note)
	}

	return nil
}

func (a Exporter) exportOneNote(ctx context.Context, note notes.Note) error {
	ankiNoteId, err := a.exportNote(ctx, note)
	if err != nil {
		return fmt.Errorf(`export note: %w`, err)
	}

	err = a.notesRepo.SetAsExported(ctx, note.ID, ankiNoteId, notes.Exported)
	if err != nil {
		return fmt.Errorf(`update status of note: %w`, err)
	}

	return nil
}

func (a Exporter) exportNote(ctx context.Context, note notes.Note) (int64, error) {

	transformedSentence := a.clozeRegex.ReplaceAllString(note.MarkedSentence, "{{c1::$1}}")

	ankiRequest_ := ankiRequest{
		Action:  "addNote",
		Version: 6,
		Params: ParamNote{
			Note: Note{
				DeckName:  a.conf.AnkiDeck,
				ModelName: a.conf.AnkiModel,
				Fields: Fields{
					Id:             fmt.Sprintf(`%d`, note.ID),
					Lemma:          note.Lemma,
					TargetWordForm: note.TargetWordForm,
					MarkedSentence: transformedSentence,
					Translation:    note.Translation,
					GrammarNote:    note.GrammarNote,
					Synonyms:       note.Synonyms,
					PartOfSpeech:   note.PartOfSpeech,
					DefinitionEn:   note.DefinitionEn,
					DefinitionRu:   note.DefinitionRu,
					TranslationRu:  note.TranslationRu,
					Audio:          ``,
					Frequency:      strconv.FormatFloat(note.Frequency, 'f', -1, 64),
				},
				Options: Options{
					AllowDuplicate: false,
					DuplicateScope: "",
					DuplicateScopeOptions: DuplicateScopeOptions{
						DeckName:       "",
						CheckChildren:  false,
						CheckAllModels: false,
					},
				},
				Tags: nil,
				Audio: []Audio{
					{
						Data:     note.AudioBase64,
						Filename: note.AudioFilename,
						SkipHash: "",
						Fields:   []string{"audio"},
					},
				},
			},
		},
	}

	response, err := a.makeRequest(ctx, ankiRequest_)
	if err != nil {
		return 0, fmt.Errorf(`make request: %w`, err)
	}

	ankiId, err := strconv.Atoi(string(response.Result))
	if err != nil {
		return 0, fmt.Errorf(`parse ankiId: %w`, err)
	}

	return int64(ankiId), nil
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
	<div style="display:none;">{{cloze:marked_sentence}}</div>
	
	<div id="raw-text" style="display:none;">{{marked_sentence}}</div>
	
	<div class="sentence-box" id="fixed-sentence"></div>
</div>
<hr id=answer>
<h3>{{definition_en}}</h3>
<div class="translation-box">
	<div class="word-translation"><b>{{lemma}}</b> — {{translation_ru}}</div>
	<div style="margin-top: 8px;">{{translation}}</div>
</div>

<script>
    var raw = document.getElementById("raw-text").innerHTML;

    raw = raw.replace(/\{\{c2::\}\}/g, "");
    
    raw = raw.replace(/\{\{c1::(.*?)\}\}/g, "<span class='cloze'>$1</span>");
 
    document.getElementById("fixed-sentence").innerHTML = raw;
</script>
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
