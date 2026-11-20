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

// contextAwareReader is a struct wrapper that will perform context cancelation checks before any IO operation is performed
type contextAwareReader struct {
	ctx context.Context
	rdr io.Reader
}

// if context has not been cancelled yet, proceed with io.Read()
func (ctxRdr *contextAwareReader) Read(p []byte) (num int, err error) {
	if err := ctxRdr.ctx.Err(); err != nil {
		return 0, err
	}
	return ctxRdr.rdr.Read(p)
}

type Storage interface {
	Save(ctx context.Context, id string, filename string, rdr io.Reader, size int64) error
	Delete(ctx context.Context, folderName string, fileName string) error
	GetFullFilePath(folderName string, fileName string) string
}

type LocalStorage struct {
	baseDir string
}

func NewLocalStorage(baseDir string) *LocalStorage {
	return &LocalStorage{baseDir: baseDir}
}

func (s *LocalStorage) Save(ctx context.Context, id string, filename string, rdr io.Reader, size int64) error {
	// check for context cancellation
	if err := ctx.Err(); err != nil {
		return err
	}

	// Create the folder
	dirPath := filepath.Join(s.baseDir, filepath.Clean("/"+id))
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return fmt.Errorf("failed to create dir: %s %w", filename, err)
	}

	// Create the file
	filePath := filepath.Join(dirPath, filepath.Clean("/"+filename))
	f, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}

	// wrap the io.Reader with a context aware reader to handle context cancellation
	ctxRdr := &contextAwareReader{
		ctx: ctx,
		rdr: rdr,
	}
	// Stream the data to the file
	if _, err := io.Copy(f, ctxRdr); err != nil {
		f.Close()
		removePartial(filePath)
		return fmt.Errorf("write file: %w", err)
	}
	if err := f.Close(); err != nil {
		removePartial(filePath)
		return fmt.Errorf("close file: %w", err)
	}
	return nil
}

func (s *LocalStorage) Delete(ctx context.Context, folderName string, fileName string) error {
	// check for context cancellation
	if err := ctx.Err(); err != nil {
		return err
	}

	filePath := s.GetFullFilePath(folderName, fileName)
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete file: %w", err)
	}
	return nil
}

func (s *LocalStorage) GetFullFilePath(folderName string, fileName string) string {
	return filepath.Join(s.baseDir, filepath.Clean("/"+folderName), filepath.Clean("/"+fileName))
}

// deletes the file truncated by a failed write
func removePartial(filePath string) {
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		log.Printf("unable to remove partial uploaded file: %s: %v", filePath, err)
	}
}
