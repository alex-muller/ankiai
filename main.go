package main

import (
	"context"
	"embed"
	"fmt"
	"log"
	"os"

	"github.com/alex-muller/ankiai/internal/config"
	"github.com/alex-muller/ankiai/internal/lib/db"
	"github.com/alex-muller/ankiai/internal/lib/logger"
	"github.com/alex-muller/ankiai/internal/module/export"
	"github.com/alex-muller/ankiai/internal/module/lexicographer"
	"github.com/alex-muller/ankiai/internal/module/notes"
	"github.com/alex-muller/ankiai/internal/module/telegram"
	"github.com/alex-muller/ankiai/internal/module/word"
	"github.com/alex-muller/ankiai/internal/run"
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

func main() {
	ctx := context.Background()

	logger.Setup(logger.EnvLocal)

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	// Conf
	conf := config.Get()

	// Db
	database, err := db.InitDB(conf.DBPath, conf.MigrationsPath)
	if err != nil {
		log.Fatalf("Не удалось инициализировать приложение: %v", err)
	}
	defer database.Close()

	// Repository
	wordRepo := word.NewRepository(database)
	notesRepo := notes.NewRepo(database)

	// Lexer
	lex := lexicographer.New(wordRepo, conf.GeminiApiKey)

	// Cards worker
	cardsWorker := notes.NewWorker(wordRepo, notesRepo)

	// TTS worker
	ttsWorker := notes.NewTtsWorker(conf, notesRepo)

	// Exporter
	exporter := export.NewExporter(conf, notesRepo)

	// Telegram
	telegramService := telegram.New(conf)

	command := os.Args[1]

	switch command {
	case "import":
		if len(os.Args) < 3 {
			printUsage()
			os.Exit(1)
		}
		// Запуск: ./ankiai import words.txt
		err = run.Import(ctx, wordRepo, os.Args[2])
		if err != nil {
			log.Fatalf(`run import "%s"`, err)
		}
	case "examples":
		run.Examples(ctx, lex)
	case "daemon":
		// Запуск: ./ankiai daemon
		run.Daemon(ctx, lex, cardsWorker, ttsWorker)
	case "export":
		run.Export(ctx, exporter)
	case "telegram":
		run.Telegram(telegramService)

	default:
		fmt.Printf("Неизвестная команда: %s\n", command)
		printUsage()
		os.Exit(1)
	}

}

func printUsage() {

	fmt.Println("Usage: ankiai <command> [arguments]")
	fmt.Println("Commands:")
	fmt.Println("  import <file>    Import words from a text file into the queue")
	fmt.Println("  export           Exports words to Anki")
	fmt.Println("  daemon           Start background processes (TG bot, workers, Anki)")
}
