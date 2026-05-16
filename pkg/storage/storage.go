package storage

import (
	"context"
	"io"
)

type Storage interface {
	Save(ctx context.Context, filePath string, reader io.Reader) error
	GetURL(filePath string) string
	Delete(ctx context.Context, filePath string) error
}
