// Package storage defines the platform file-storage contract.
package storage

import (
	"context"
	"time"
)

// Storage defines an interface for interacting with the file storage system.
type Storage interface {
	// UploadURL returns a URL the client can PUT a file to directly.
	UploadURL(ctx context.Context, path string, expiry time.Duration) (string, error)
	// DownloadURL returns a URL the client can GET a file from directly.
	DownloadURL(ctx context.Context, path string, expiry time.Duration) (string, error)
	// Delete deletes file by path.
	Delete(ctx context.Context, path string) error
	// Move relocates a file from src to dst within the same backend.
	Move(ctx context.Context, src, dst string) error
}
