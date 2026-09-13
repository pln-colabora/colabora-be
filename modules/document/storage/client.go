package storage

import (
	"context"
	"io"
)

// Client is the narrow storage interface the document service needs — kept small and
// interface-backed so the service is unit-testable without a live Garage instance.
type Client interface {
	PutObject(ctx context.Context, key string, r io.Reader, size int64, contentType string) error
	DeleteObject(ctx context.Context, key string) error
	GetObject(ctx context.Context, key string) (io.ReadCloser, error)
}
