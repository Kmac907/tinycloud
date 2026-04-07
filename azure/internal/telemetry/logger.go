package telemetry

import (
	"encoding/json"
	"io"
	"sync"
	"time"
)

type Logger struct {
	out io.Writer
	mu  sync.Mutex
}

func NewJSONLogger(out io.Writer) *Logger {
	return &Logger{out: out}
}

func (l *Logger) Info(message string, fields map[string]any) {
	l.write("info", message, fields)
}

func (l *Logger) Error(message string, fields map[string]any) {
	l.write("error", message, fields)
}

func (l *Logger) write(level, message string, fields map[string]any) {
	record := map[string]any{
		"ts":      time.Now().UTC().Format(time.RFC3339Nano),
		"level":   level,
		"message": message,
	}
	for key, value := range fields {
		record[key] = value
	}

	body, _ := json.Marshal(record)
	l.mu.Lock()
	defer l.mu.Unlock()
	_, _ = l.out.Write(append(body, '\n'))
}
