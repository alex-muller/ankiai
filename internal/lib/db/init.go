package db

import (
	"embed"
	"fmt"
	"log"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3" // Важно: анонимный импорт драйвера
	"github.com/pressly/goose/v3"
)

// InitDB открывает соединение и автоматически применяет миграции
func InitDB(dbPath string, embedMigrations embed.FS) (*sqlx.DB, error) {
	// Подключаемся к SQLite. Если файла dbPath нет, драйвер создаст его сам.
	db, err := sqlx.Connect("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("ошибка подключения к БД: %w", err)
	}

	// Настраиваем goose для использования встроенной файловой системы
	goose.SetBaseFS(embedMigrations)

	// Указываем goose, с каким диалектом работаем
	if err := goose.SetDialect("sqlite3"); err != nil {
		return nil, fmt.Errorf("ошибка установки диалекта goose: %w", err)
	}

	// Накатываем миграции из папки "migrations"
	log.Println("Проверка и применение миграций базы данных...")
	if err := goose.Up(db.DB, "migrations"); err != nil {
		return nil, fmt.Errorf("ошибка применения миграций: %w", err)
	}
	log.Println("Миграции успешно применены!")

	return db, nil
}
