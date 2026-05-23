package notes

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/alex-muller/ankiai/internal/config"
)

func NewTtsWorker(conf config.Config, repo *Repo) *Tts {
	return &Tts{
		ttsApiKey: conf.TtsApiKey,
		repo:      repo,
	}
}

type Tts struct {
	ttsApiKey string
	repo      *Repo
}

func (a Tts) Run() {

}

func (a Tts) runOnce(ctx context.Context) error {
	notes, err := a.repo.GetManyByStatus(ctx, StatusFrequencyAdded, 1)
	if err != nil {
		return fmt.Errorf(`get notes: %w`, err)
	}

	ch := make(chan Note)

	go func() {
		for _, note := range notes {
			select {
			case <-ctx.Done():
				return
			default:
				ch <- note
			}
		}
	}()

	wg := &sync.WaitGroup{}

	for i := 10; i > 0; i-- {
		wg.Add(1)
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case note := <-ch:
					err_ := a.processOneNote(ctx, note)
					if err_ != nil {
						// TODO finish this
					}
				}
			}
		}()
	}
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

	url := ` https://texttospeech.googleapis.com/v1/text:synthesize?key=` + a.ttsApiKey

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
