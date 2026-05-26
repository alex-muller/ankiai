package run

import (
	"context"
	"fmt"
	"strings"

	"github.com/alex-muller/ankiai/internal/module/importer"
)

func ImportFromFile(ctx context.Context, importer *importer.Service, filePath string) error {
	imported, err := importer.ImportFromFile(ctx, filePath)
	if err != nil {
		return err
	}
	if len(imported) > 0 {
		fmt.Printf("Imported from file: [%s]\n", strings.Join(imported, ", "))
	} else {
		fmt.Printf("Nothing to import\n")
	}
	return nil
}
