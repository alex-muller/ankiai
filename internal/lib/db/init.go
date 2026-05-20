package db

import (
	"fmt"
	"os"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3" // Важно: анонимный импорт драйвера
	"github.com/pressly/goose/v3"
)

// InitDB открывает соединение и автоматически применяет миграции
func InitDB(dbPath, migrationPath string) (*sqlx.DB, error) {
	// Подключаемся к SQLite. Если файла dbPath нет, драйвер создаст его сам.
	db, err := sqlx.Connect("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("db connect error: %w", err)
	}

	// Настраиваем goose для использования встроенной файловой системы
	fsys := os.DirFS(migrationPath)
	goose.SetBaseFS(fsys)

	// Указываем goose, с каким диалектом работаем
	if err := goose.SetDialect("sqlite3"); err != nil {
		return nil, fmt.Errorf("goose error: %w", err)
	}

	// Накатываем миграции из папки "migrations"
	if err := goose.Up(db.DB, "."); err != nil {
		return nil, fmt.Errorf("migration error: %w", err)
	}

	return db, nil
}
