package storage

import (
	"context"
	"io"
	"os"
	"path/filepath"
)

type LocalStorage struct {
	baseDir string
	baseURL string
}

func NewLocalStorage(baseDir, baseURL string) *LocalStorage {
	return &LocalStorage{baseDir: baseDir, baseURL: baseURL}
}

func (s *LocalStorage) Save(_ context.Context, filePath string, reader io.Reader) error {
	fullPath := filepath.Join(s.baseDir, filePath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return err
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		return err
	}
	return os.WriteFile(fullPath, data, 0644)
}

func (s *LocalStorage) GetURL(filePath string) string {
	return s.baseURL + "/" + filePath
}

func (s *LocalStorage) Delete(_ context.Context, filePath string) error {
	return os.Remove(filepath.Join(s.baseDir, filePath))
}
