package notes

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/alex-muller/ankiai/internal/config"
	"github.com/alex-muller/ankiai/internal/lib/logger"
)

func NewTtsWorker(conf config.Config, repo *Repo) *Tts {
	return &Tts{
		ttsApiKey: conf.TtsApiKey,
		repo:      repo,
		log:       logger.Logger.With(slog.String("component", "tts")),
	}
}

type Tts struct {
	ttsApiKey string
	repo      *Repo
	log       *slog.Logger
}

func (a Tts) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			a.runOnce(ctx)
			time.Sleep(2100 * time.Millisecond)
		}
	}
}

func (a Tts) runOnce(ctx context.Context) {
	ch := make(chan Note)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	wg := &sync.WaitGroup{}

	for i := 2; i > 0; i-- {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case note, ok := <-ch:
					if !ok {
						return
					}
					err_ := a.processOneNote(ctx, note)
					if err_ != nil {
						a.log.Error(`process one note`, slog.String(`error`, err_.Error()))
					}
					fmt.Println(fmt.Sprintf("got mp3 for note: %s - %s", note.TargetWordForm, note.CardHash))
					time.Sleep(2100 * time.Millisecond)
				}
			}
		}()
	}

	notes, err := a.repo.GetManyByStatus(ctx, GenerateAudioPending, 10)
	if err != nil {
		a.log.Error(`get notes`, slog.String(`error`, err.Error()))
		return
	}

	for _, note := range notes {
		select {
		case <-ctx.Done():
			return
		default:
			ch <- note
		}
	}

	close(ch)

	wg.Wait()
}

func (a Tts) processOneNote(ctx context.Context, note Note) error {
	audio, err := a.getPhrase(ctx, note.MarkedSentence)
	if err != nil {
		return fmt.Errorf(`get phrase: %w`, err)
	}

	err = a.repo.AddAudio(ctx, note.ID, audio, note.CardHash+`.mp3`)
	if err != nil {
		return fmt.Errorf(`add audio: %w`, err)
	}

	return nil
}

func (a Tts) getPhrase(ctx context.Context, phrase string) (string, error) {
	phrase = strings.ReplaceAll(phrase, "*", "")

	payload := &TtsPayload{
		Input: struct {
			Text string `json:"text"`
		}{
			Text: phrase,
		},
		Voice: struct {
			LanguageCode string `json:"languageCode"`
			Name         string `json:"name"`
		}{
			LanguageCode: "en-US",
			Name:         "en-US-Journey-F",
		},
		AudioConfig: struct {
			AudioEncoding string `json:"audioEncoding"`
		}{
			AudioEncoding: "MP3",
		},
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf(`payload marshal to json: %w`, err)
	}

	url := "https://texttospeech.googleapis.com/v1/text:synthesize?key=" + a.ttsApiKey

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return ``, fmt.Errorf(`make request: %w`, err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return ``, fmt.Errorf(`client do: %w`, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ``, fmt.Errorf(`read body: %w`, err)
	}

	if resp.StatusCode != http.StatusOK {
		return ``, fmt.Errorf(`unexpected status code: %d, body: %s`, resp.StatusCode, body)
	}

	var respJson TtsResponse

	err = json.Unmarshal(body, &respJson)
	if err != nil {
		return ``, fmt.Errorf(`json unmarshal: %w`, err)
	}

	return respJson.AudioContent, nil
}

type TtsPayload struct {
	Input struct {
		Text string `json:"text"`
	} `json:"input"`
	Voice struct {
		LanguageCode string `json:"languageCode"`
		Name         string `json:"name"`
	} `json:"voice"`
	AudioConfig struct {
		AudioEncoding string `json:"audioEncoding"`
	} `json:"audioConfig"`
}

type TtsResponse struct {
	AudioContent string `json:"audioContent"`
}
