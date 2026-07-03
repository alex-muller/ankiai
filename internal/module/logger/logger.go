package logger

import (
	"fmt"
	"sync"
)

var l *multiLineLogger
var mu sync.Mutex

type multiLineLogger struct {
	mu    sync.Mutex
	lines []string
}

func NewLogger(prefix string) *Logger {
	mu.Lock()
	if l == nil {
		l = &multiLineLogger{}
	}

	l.lines = append(l.lines, ``)
	mu.Unlock()
	fmt.Println()
	return &Logger{
		prefix: prefix,
		line:   len(l.lines) - 1,
		m:      l,
	}
}

// Обновление конкретной строки
func (ml *multiLineLogger) updateLine(lineIdx int, text string) {
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

type Logger struct {
	prefix string
	line   int

	m *multiLineLogger
}

func (a *Logger) Log(message string) {
	a.m.updateLine(a.line, fmt.Sprintf(`[%s] %s`, a.prefix, message))
}
