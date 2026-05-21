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

	return err
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

	_, ok := resp.Result[name]
	return ok, nil
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
}

type ankiResponse struct {
	Result map[string]int64 `json:"result"`
	Error  interface{}      `json:"error"`
}
