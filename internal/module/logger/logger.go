package logger

import (
	"fmt"
	"sync"
)

type MultiLineLogger struct {
	mu    sync.Mutex
	lines []string
}

func NewLogger(lineCount int) *MultiLineLogger {
	// Инициализируем пустые строки, чтобы сразу занять место в терминале
	lines := make([]string, lineCount)
	for i := 0; i < lineCount; i++ {
		fmt.Println()
	}
	return &MultiLineLogger{lines: lines}
}

// Обновление конкретной строки
func (ml *MultiLineLogger) UpdateLine(lineIdx int, text string) {
	ml.mu.Lock()
	defer ml.mu.Unlock()

	ml.lines[lineIdx] = text

	// 1. Поднимаем курсор в самый верх нашего блока строк
	totalLines := len(ml.lines)
	fmt.Printf("\033[%dF", totalLines)

	// 2. Перезаписываем каждую строку заново
	for _, line := range ml.lines {
		// \r — в начало, \033[K — очистить старый хвост, если новая строка короче
		fmt.Printf("\r\033[K%s\n", line)
	}
}
