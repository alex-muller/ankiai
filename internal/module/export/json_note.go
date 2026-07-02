package export

type ParamNote struct {
	Note Note `json:"note,omitempty"`
}

type Note struct {
	DeckName  string    `json:"deckName,omitempty"`
	ModelName string    `json:"modelName,omitempty"`
	Fields    Fields    `json:"fields,omitempty"`
	Options   Options   `json:"options,omitempty"`
	Tags      []string  `json:"tags,omitempty"`
	Audio     []Audio   `json:"audio,omitempty"`
	Video     []Video   `json:"video,omitempty"`
	Picture   []Picture `json:"picture,omitempty"`
}

type Fields struct {
	Id             string `json:"id,omitempty"`
	Lemma          string `json:"lemma,omitempty"`
	TargetWordForm string `json:"target_word_form,omitempty"`
	MarkedSentence string `json:"marked_sentence,omitempty"`
	Translation    string `json:"translation,omitempty"`
	GrammarNote    string `json:"grammar_note,omitempty"`
	Synonyms       string `json:"synonyms,omitempty"`
	PartOfSpeech   string `json:"part_of_speech,omitempty"`
	DefinitionEn   string `json:"definition_en,omitempty"`
	DefinitionRu   string `json:"definition_ru,omitempty"`
	TranslationRu  string `json:"translation_ru,omitempty"`
	DefinitionPl   string `json:"definition_pl,omitempty"`
	TranslationPl  string `json:"translation_pl,omitempty"`
	Audio          string `json:"audio,omitempty"`
	AudioPl        string `json:"audio_pl,omitempty"`
	Frequency      string `json:"frequency,omitempty"`
}

type DuplicateScopeOptions struct {
	DeckName       string `json:"deckName,omitempty"`
	CheckChildren  bool   `json:"checkChildren,omitempty"`
	CheckAllModels bool   `json:"checkAllModels,omitempty"`
}

type Options struct {
	AllowDuplicate        bool                  `json:"allowDuplicate,omitempty"`
	DuplicateScope        string                `json:"duplicateScope,omitempty"`
	DuplicateScopeOptions DuplicateScopeOptions `json:"duplicateScopeOptions,omitempty"`
}

type Audio struct {
	Data     string   `json:"data,omitempty"`
	Url      string   `json:"url,omitempty"`
	Filename string   `json:"filename,omitempty"`
	SkipHash string   `json:"skipHash,omitempty"`
	Fields   []string `json:"fields,omitempty"`
}

type Video struct {
	Url      string   `json:"url,omitempty"`
	Filename string   `json:"filename,omitempty"`
	SkipHash string   `json:"skipHash,omitempty"`
	Fields   []string `json:"fields,omitempty"`
}

type Picture struct {
	Url      string `json:"url,omitempty"`
	Filename string `json:"filename,omitempty"`
	SkipHash string `json:"skipHash,omitempty"`
}
