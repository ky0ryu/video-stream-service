// Package storage
package storage

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
)

type Storage interface {
	Save(ctx context.Context, key string, r io.Reader, size int64) error
	Delete(ctx context.Context, key string) error
}

type LocalStorage struct {
	baseDir string
}

func NewLocalStorage(baseDir string) *LocalStorage {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		log.Fatalf("failed to create dir: %v %v", baseDir, err)
	}
	return &LocalStorage{baseDir: baseDir}
}

func (s *LocalStorage) Save(ctx context.Context, filename string, r io.Reader, size int64) error {
	dest := filepath.Join(s.baseDir, filepath.Clean("/"+filename))

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("ctx error found: %w", err)
	}

	// Create the file
	f, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()

	// Stream the data to the file
	if _, err := io.Copy(f, r); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}

func (s *LocalStorage) Delete(ctx context.Context, filename string) error {
	dest := filepath.Join(s.baseDir, filepath.Clean("/"+filename))
	if err := os.Remove(dest); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete file: %w", err)
	}
	return nil
}
