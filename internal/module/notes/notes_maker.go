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

	"github.com/alex-muller/ankiai/internal/lib"
	"github.com/alex-muller/ankiai/internal/lib/logger"
	"github.com/alex-muller/ankiai/internal/module/word"
)

func NewMaker(
	notesRepo *Repo,
	wordsRepo *word.Repository,
) *Maker {
	return &Maker{
		notesRepo: notesRepo,
		wordsRepo: wordsRepo,
		log:       logger.Logger.With(slog.String("component", "notes-maker")),
	}
}

type Maker struct {
	notesRepo *Repo
	wordsRepo *word.Repository
	log       *slog.Logger
}

func (a Maker) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			err := a.run(ctx)
			if err != nil {
				a.log.Error(`run`, slog.String("error", err.Error()))
			}
			fmt.Println(`--- card maker done, sleep ---`)
			time.Sleep(time.Minute)
		}
	}

}

func (a Maker) run(ctx context.Context) error {
	addedWords, err := a.wordsRepo.GetByStatus(ctx, word.StatusRaw)
	if err != nil {
		return fmt.Errorf(`get words: %w`, err)
	}

	for _, addedWord := range addedWords {
		err = a.processWord(ctx, addedWord)
		if err != nil {
			a.log.Error(`run`, slog.String("error", fmt.Errorf(`process word [%s], error: %w`, addedWord.Word, err).Error()))
		}
	}
	return nil
}

func (a Maker) processWord(ctx context.Context, word_ word.Word) error {
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

func (a Maker) processCardJson(ctx context.Context, cardJson CardJson, word_ word.Word) error {
	now := time.Now()
	for _, sens := range cardJson.Senses {
		for _, example := range sens.Examples {
			sum := md5.Sum([]byte(example.MarkedSentence))
			ankiCard := Note{
				WordID:         word_.ID,
				Lemma:          strings.ToLower(cardJson.Lemma),
				CardHash:       hex.EncodeToString(sum[:]),
				TargetWordForm: lib.CleanWord(example.TargetWordForm),
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

			err := a.notesRepo.Add(ctx, ankiCard)

			if err != nil {
				if strings.Contains(err.Error(), "UNIQUE constraint failed") {
					continue
				}
				return fmt.Errorf(`add note [%+v], error: %w`, ankiCard, err)
			}
		}
	}

	return nil
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
