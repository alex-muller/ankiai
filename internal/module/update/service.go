package update

import (
	"context"

	"github.com/alex-muller/ankiai/internal/module/frequency"
)

type Service struct {
	freq frequency.FrequencyService
}

func (a Service) Run(ctx context.Context) error {
	return nil
}

func (a Service) UpdateFreq() {

}
