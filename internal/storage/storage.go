package storage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

type Storage interface {
	Save(filename string, reader io.Reader) (string, error)
	Get(path string) (io.ReadCloser, error)
	Delete(path string) error
	GetFullPath(storagePath string) string
}

type LocalStorage struct {
	basePath string
}

func NewLocalStorage(basePath string) (*LocalStorage, error) {
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}
	return &LocalStorage{basePath: basePath}, nil
}

func (s *LocalStorage) Save(originalFilename string, reader io.Reader) (string, error) {
	ext := filepath.Ext(originalFilename)
	filename := uuid.New().String() + ext
	
	storagePath := filepath.Join(s.basePath, filename)
	
	file, err := os.Create(storagePath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	if _, err := io.Copy(file, reader); err != nil {
		os.Remove(storagePath)
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return filename, nil
}

func (s *LocalStorage) Get(storagePath string) (io.ReadCloser, error) {
	fullPath := filepath.Join(s.basePath, storagePath)
	return os.Open(fullPath)
}

func (s *LocalStorage) Delete(storagePath string) error {
	fullPath := filepath.Join(s.basePath, storagePath)
	return os.Remove(fullPath)
}

func (s *LocalStorage) GetFullPath(storagePath string) string {
	return filepath.Join(s.basePath, storagePath)
}
