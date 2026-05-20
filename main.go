package main

import (
	"embed"
	"log"

	"github.com/alex-muller/ankiai/internal/lib/db"
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

func main() {
	// Инициализируем базу в локальном файле
	// В продакшене путь лучше брать из переменной окружения или конфига
	database, err := db.InitDB("./tmp/anki_vocabulary.db", embedMigrations)
	if err != nil {
		log.Fatalf("Не удалось инициализировать приложение: %v", err)
	}
	defer database.Close()

	log.Println("Демон запущен, база данных готова к работе.")

	// Здесь стартуют ваши воркеры (Telegram-бот, Gemini, TTS, Anki)
	// go startGenerators(database)
	// go startAnkiSyncer(database)

	// Блокируем main горутину
	select {}
}
