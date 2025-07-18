package storage

import "context"

type Storage interface {
	Save(ctx context.Context, path string, data []byte) error
	Load(ctx context.Context, path string) ([]byte, error)
	List(ctx context.Context, pattern string) ([]string, error)
}