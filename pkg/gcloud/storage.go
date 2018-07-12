package gcloud

import (
	"context"

	"cloud.google.com/go/storage"
)

// NewStorageClient creates a new Google Cloud API client
func NewStorageClient() (*storage.Client, error) {
	return storage.NewClient(context.Background())
}
