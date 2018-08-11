package gcloud

import (
	"context"
	"net/http"
	"time"

	"cloud.google.com/go/storage"
	"github.com/spf13/viper"
	"github.com/stateshape/augur-analyzer/pkg/env"
	"google.golang.org/api/option"
	ghttp "google.golang.org/api/transport/http"
)

const (
	MaxIdleConns        = 100
	MaxIdleConnsPerHost = 50
)

type WriteObjectParameters struct {
	Bucket     string
	ObjectName string
	Content    []byte
}

// NewStorageClient creates a new Google Cloud API client
func NewStorageClient() (*storage.Client, error) {
	// Assembles options for custom client
	options := []option.ClientOption{option.WithScopes(storage.ScopeFullControl)}
	if viper.GetString(env.GoogleApplicationCredentials) != "" {
		options = append(options, option.WithCredentialsFile(viper.GetString(env.GoogleApplicationCredentials)))
	}

	// Create transport
	transport, err := ghttp.NewTransport(context.TODO(), &http.Transport{
		MaxIdleConns:        MaxIdleConns,
		MaxIdleConnsPerHost: MaxIdleConnsPerHost,
		IdleConnTimeout:     time.Minute,
	}, options...)
	if err != nil {
		return nil, err
	}

	// Create client with custom HTTP transport
	return storage.NewClient(context.Background(), option.WithHTTPClient(&http.Client{
		Transport: transport,
	}))
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
