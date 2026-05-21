package export

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func NewExporter() *Exporter {
	return &Exporter{}
}

type Exporter struct{}

func (a Exporter) Run(ctx context.Context) error {
	err := a.checkAndCreateDeck(ctx)

	return err
}

func (a Exporter) checkAndCreateDeck(ctx context.Context) error {
	ankiRequest_ := ankiRequest{
		Action:  "deckNamesAndIds",
		Version: 6,
	}

	return a.makeRequest(ctx, ankiRequest_)

}

func (a Exporter) makeRequest(ctx context.Context, request ankiRequest) error {
	payload, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf(`json marshal: %w`, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://127.0.0.1:8765", bytes.NewBuffer(payload)) // TODO url to env
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)

	var response ankiResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("unmarshal response: %w", err)
	}

	return nil
}

type ankiRequest struct {
	Action  string `json:"action"`
	Version int    `json:"version"`
}

type ankiResponse struct {
	Result map[string]int64 `json:"result"`
	Error  interface{}      `json:"error"`
}
