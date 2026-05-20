package main

import (
	"embed"
	"fmt"
	"log"
	"os"

	"github.com/alex-muller/ankiai/internal/lib/db"
	"github.com/alex-muller/ankiai/internal/run"
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	// Init DB
	database, err := db.InitDB("./tmp/anki_vocabulary.db", embedMigrations)
	if err != nil {
		log.Fatalf("Не удалось инициализировать приложение: %v", err)
	}
	defer database.Close()

	command := os.Args[1]

	switch command {
	case "import":
		if len(os.Args) < 3 {
			printUsage()
			os.Exit(1)
		}
		// Запуск: ./ankiai import words.txt
		run.RunImport(database, os.Args[2])

	case "daemon":
		// Запуск: ./ankiai daemon
		run.RunDaemon(database)

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
	fmt.Println("  daemon           Start background processes (TG bot, workers, Anki)")
}
