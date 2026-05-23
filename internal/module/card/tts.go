package card

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/alex-muller/ankiai/internal/config"
)

func NewTtsWorker(conf config.Config) *Tts {
	return &Tts{
		ttsApiKey: conf.TtsApiKey,
	}
}

type Tts struct {
	ttsApiKey string
}

func (a Tts) Run() {

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
