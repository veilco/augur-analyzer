package gcloud

import (
	"bytes"
	"compress/gzip"

	"cloud.google.com/go/storage"
	"github.com/golang/protobuf/proto"
)

type UploadObjectRequest struct {
	Error  chan error
	Object *UploadObject
}

type UploadObject struct {
	Msg            proto.Message
	Bucket         string
	Object         string
	WriterModifier func(*storage.Writer)
	IsGZIP         bool
}

type ObjectUploader struct {
	storage *storage.Client
	workers chan chan *UploadObjectRequest
}

func NewObjectUploader() (*ObjectUploader, error) {
	client, err := NewStorageClient()
	if err != nil {
		return nil, err
	}

	// Start workers
	workers := make(chan chan *UploadObjectRequest, MaxIdleConns)
	for i := 0; i < MaxIdleConns; i++ {
		worker := NewUploadWorker(client, workers)
		go worker.Start()
	}

	return &ObjectUploader{
		storage: client,
		workers: workers,
	}, nil
}

func (ou *ObjectUploader) WriteObject(object *UploadObject) error {
	errchan := make(chan error, 1)
	<-ou.workers <- &UploadObjectRequest{
		Error:  errchan,
		Object: object,
	}
	for err := range errchan {
		return err
	}
	return nil
}

type UploadWorker struct {
	Inbox   chan *UploadObjectRequest
	Workers chan chan *UploadObjectRequest

	storage *storage.Client
}

func NewUploadWorker(client *storage.Client, workers chan chan *UploadObjectRequest) *UploadWorker {
	return &UploadWorker{
		Inbox:   make(chan *UploadObjectRequest, 1),
		Workers: workers,
		storage: client,
	}
}

func (uw *UploadWorker) Start() {
	uw.Workers <- uw.Inbox

	for upload := range uw.Inbox {
		uw.ProcessUpload(upload)
		uw.Workers <- uw.Inbox
	}
}

func (uw *UploadWorker) ProcessUpload(request *UploadObjectRequest) {
	defer close(request.Error)

	var (
		content       = []byte{}
		err     error = nil
	)

	content, err = proto.Marshal(request.Object.Msg)
	if err != nil {
		request.Error <- err
		return
	}

	if request.Object.IsGZIP {
		gzipped := bytes.NewBuffer(nil)
		gwrtr := gzip.NewWriter(gzipped)
		if _, err := gwrtr.Write(content); err != nil {
			request.Error <- err
			return
		}
		if err := gwrtr.Close(); err != nil {
			request.Error <- err
			return
		}
		content = gzipped.Bytes()
	}
	if err := WriteObject(uw.storage, WriteObjectParameters{
		Bucket:     request.Object.Bucket,
		ObjectName: request.Object.Object,
		Content:    content,
	}, request.Object.WriterModifier); err != nil {
		request.Error <- err
		return
	}

	return
}
