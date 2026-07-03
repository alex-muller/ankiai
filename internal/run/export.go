package run

import (
	"context"

	"github.com/alex-muller/ankiai/internal/module/export"
)

func Export(ctx context.Context, exporter *export.Exporter) {
	err := exporter.Run(ctx)
	if err != nil {
		panic(err)
	}
}

func UpdateNotes(ctx context.Context, exporter *export.Exporter) {
	err := exporter.Update(ctx)
	if err != nil {
		panic(err)
	}
}
