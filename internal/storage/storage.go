package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type Storage interface {
	Save(ctx context.Context, key string, r io.Reader) error
	Open(ctx context.Context, key string) (io.ReadCloser, error)
	Remove(ctx context.Context, key string) error
}

type Local struct {
	Dir string
}

func NewLocal(dir string) (*Local, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create storage dir: %w", err)
	}
	return &Local{Dir: dir}, nil
}

func (l *Local) path(key string) string {
	// filepath.Base strips any directory components, guarding against traversal.
	return filepath.Join(l.Dir, filepath.Base(key))
}

func (l *Local) Save(ctx context.Context, key string, r io.Reader) error {
	f, err := os.Create(l.path(key))
	if err != nil {
		return fmt.Errorf("storage save: %w", err)
	}
	defer f.Close()
	if _, err := io.Copy(f, r); err != nil {
		return fmt.Errorf("storage save copy: %w", err)
	}
	return nil
}

func (l *Local) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	f, err := os.Open(l.path(key))
	if err != nil {
		return nil, fmt.Errorf("storage open: %w", err)
	}
	return f, nil
}

func (l *Local) Remove(ctx context.Context, key string) error {
	return os.Remove(l.path(key))
}
