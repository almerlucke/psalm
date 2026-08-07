package logger

import (
	"log/slog"

	"github.com/almerlucke/psalm"
)

type Logger struct {
	*psalm.Module
	logger *slog.Logger
}

func New(logger *slog.Logger) *Logger {
	l := &Logger{
		Module: psalm.NewControl(),
		logger: logger,
	}
	l.SetSelf(l)
	return l
}

func (l *Logger) Receive(value any, index int) {
	l.logger.Info("Logger received", "id", l.ID(), "value", value, "index", index)
}
