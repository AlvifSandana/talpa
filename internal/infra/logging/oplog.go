package logging

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"talpa/internal/domain/model"
)

type Logger interface {
	Log(ctx context.Context, entry model.OperationLogEntry) error
}

type noopLogger struct{}

func (n noopLogger) Log(context.Context, model.OperationLogEntry) error { return nil }

func NewNoopLogger() Logger { return noopLogger{} }

type operationLogger struct {
	mu   sync.Mutex
	file *os.File
}

func NewOperationLogger(ctx context.Context, disabled bool) (Logger, error) {
	if disabled {
		return noopLogger{}, nil
	}

	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		configHome = filepath.Join(home, ".config")
	}

	dir := filepath.Join(configHome, "talpa")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}

	f, err := os.OpenFile(filepath.Join(dir, "operations.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, err
	}

	_ = ctx
	return &operationLogger{file: f}, nil
}

func (l *operationLogger) Log(_ context.Context, entry model.OperationLogEntry) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().UTC()
	}

	b, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	_, err = l.file.Write(append(b, '\n'))
	return err
}
