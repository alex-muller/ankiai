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
	"github.com/alex-muller/ankiai/internal/module/lexicographer"
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
	repository := word.NewRepository(database)

	// Lexer
	lex := lexicographer.New(repository, conf.GeminiApiKey)

	command := os.Args[1]

	switch command {
	case "import":
		if len(os.Args) < 3 {
			printUsage()
			os.Exit(1)
		}
		// Запуск: ./ankiai import words.txt
		err = run.Import(ctx, repository, os.Args[2])
		if err != nil {
			log.Fatalf(`run import "%s"`, err)
		}

	case "daemon":
		// Запуск: ./ankiai daemon
		run.Daemon(ctx, lex)

	default:
		fmt.Printf("Неизвестная команда: %s\n", command)
		printUsage()
		os.Exit(1)
	}

}

func printUsage() {
	l := logger.Logger
	l.Info("Usage: ankiai <command> [arguments]")
	l.Info("Commands:")
	l.Info("  import <file>    Import words from a text file into the queue")
	l.Info("  daemon           Start background processes (TG bot, workers, Anki)")
}
