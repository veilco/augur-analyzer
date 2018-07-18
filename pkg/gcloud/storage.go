package gcloud

import (
	"context"

	"cloud.google.com/go/storage"
)

type WriteObjectParameters struct {
	Bucket     string
	ObjectName string
	Content    []byte
}

// NewStorageClient creates a new Google Cloud API client
func NewStorageClient() (*storage.Client, error) {
	return storage.NewClient(context.Background())
}

func WriteObject(client *storage.Client, params WriteObjectParameters, modifiers ...func(wrtr *storage.Writer)) error {
	bkt := client.Bucket(params.Bucket)
	obj := bkt.Object(params.ObjectName)
	wrtr := obj.NewWriter(context.Background())
	for _, m := range modifiers {
		m(wrtr)
	}
	if _, err := wrtr.Write(params.Content); err != nil {
		return err
	}
	if err := wrtr.Close(); err != nil {
		return err
	}
	return nil
}
