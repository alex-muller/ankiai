package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

// Config holds application configuration.
type Config struct {
	DBPath           string
	MigrationsPath   string
	GeminiApiKey     string
	TtsApiKey        string
	AnkiConnectUrl   string
	AnkiDeck         string
	AnkiModel        string
	TelegramBotToken string
}

// Get loads and returns the application configuration.
func Get() Config {
	// Загружаем .env файл
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found or could not be loaded: %v", err)
	}

	return Config{
		DBPath:           `tmp/anki_vocabulary.dev.db`,
		MigrationsPath:   `migrations`,
		GeminiApiKey:     os.Getenv("GEMINI_API_KEY"),
		TtsApiKey:        os.Getenv("GOOGLE_TTS_API_KEY"),
		AnkiConnectUrl:   os.Getenv("ANKI_CONNECT_URL"),
		AnkiDeck:         os.Getenv("ANKI_DECK_NAME"),
		AnkiModel:        os.Getenv("ANKI_MODEL_NAME"),
		TelegramBotToken: os.Getenv("TELEGRAM_BOT_TOKEN"),
	}
}
