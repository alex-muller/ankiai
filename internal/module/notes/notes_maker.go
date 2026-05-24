package notes

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/alex-muller/ankiai/internal/lib/logger"
	"github.com/alex-muller/ankiai/internal/module/word"
)

func newMaker(
	cardRepo *Repo,
	wordsRepo *word.Repository,
) *maker {
	return &maker{
		cardRepo:  cardRepo,
		wordsRepo: wordsRepo,
		log:       logger.Logger.With(slog.String("component", "card-maker")),
	}
}

type maker struct {
	cardRepo  *Repo
	wordsRepo *word.Repository
	log       *slog.Logger
}

func (a maker) Run(ctx context.Context) {
	err := a.run(ctx)
	if err != nil {
		a.log.Error(`run`, slog.String("error", err.Error()))
	}
}

func (a maker) run(ctx context.Context) error {
	addedWords, err := a.wordsRepo.GetByStatus(ctx, word.StatusRaw)
	if err != nil {
		return fmt.Errorf(`get words: %w`, err)
	}

	for _, addedWord := range addedWords {
		err = a.processWord(ctx, addedWord)
		if err != nil {
			return fmt.Errorf(`process word [%s], error: %w`, addedWord.Word, err)
		}
	}
	return nil
}

func (a maker) processWord(ctx context.Context, word_ word.Word) error {
	resp := GeminiResponse{}

	err := json.Unmarshal([]byte(word_.RawJSON), &resp)
	if err != nil {
		return fmt.Errorf(`json decode raw: %w`, err)
	}

	var cardJson CardJson

	for _, candidate := range resp.Candidates {
		for _, part := range candidate.Content.Parts {
			err = json.Unmarshal([]byte(part.Text), &cardJson)
			if err != nil {
				return fmt.Errorf(`json decode part: %w`, err)
			}
		}
	}

	if cardJson.Lemma == `` {
		return nil
	}

	err = a.processCardJson(ctx, cardJson, word_)
	if err != nil {
		return fmt.Errorf(`process card: %w`, err)
	}

	word_.Status = word.StatusCardsCreated
	err = a.wordsRepo.Update(ctx, word_)
	if err != nil {
		return fmt.Errorf(`update word: %w`, err)
	}

	return nil
}

func (a maker) processCardJson(ctx context.Context, cardJson CardJson, word_ word.Word) error {
	now := time.Now()
	for _, sens := range cardJson.Senses {
		for _, example := range sens.Examples {
			sum := md5.Sum([]byte(example.MarkedSentence))
			ankiCard := Note{
				WordID:         word_.ID,
				Lemma:          strings.ToLower(cardJson.Lemma),
				CardHash:       hex.EncodeToString(sum[:]),
				TargetWordForm: a.cleanWord(example.TargetWordForm),
				MarkedSentence: example.MarkedSentence,
				Translation:    example.Translation,
				GrammarNote:    example.GrammarNote,
				Synonyms:       strings.Join(sens.Synonyms, ", "),
				PartOfSpeech:   strings.ToLower(sens.PartOfSpeech),
				DefinitionEn:   sens.DefinitionEn,
				DefinitionRu:   sens.DefinitionRu,
				TranslationRu:  example.Translation,
				AudioFilename:  "",
				AudioBase64:    "",
				Status:         0,
				CreatedAt:      now,
			}

			err := a.cardRepo.Add(ctx, ankiCard)

			a.log.Info(
				`added card`,
				slog.String(`lemma`, ankiCard.Lemma),
				slog.String(`word`, ankiCard.TargetWordForm),
			)

			if err != nil {
				return fmt.Errorf(`add card: %w`, err)
			}
		}
	}

	return nil
}

func (a maker) cleanWord(text string) string {
	text = strings.ToLower(text)
	return text
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

type CardJson struct {
	Lemma  string `json:"lemma"`
	Senses []struct {
		PartOfSpeech  string   `json:"part_of_speech"`
		TranslationRu string   `json:"translation_ru"`
		DefinitionEn  string   `json:"definition_en"`
		DefinitionRu  string   `json:"definition_ru"`
		Synonyms      []string `json:"synonyms"`
		Examples      []struct {
			GrammarNote    string `json:"grammar_note"`
			TargetWordForm string `json:"target_word_form"`
			MarkedSentence string `json:"marked_sentence"`
			Translation    string `json:"translation"`
		} `json:"examples"`
	} `json:"senses"`
}
