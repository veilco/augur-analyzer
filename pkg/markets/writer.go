package markets

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"strings"

	"github.com/stateshape/augur-analyzer/pkg/gcloud"
	"github.com/stateshape/augur-analyzer/pkg/proto/markets"

	"cloud.google.com/go/storage"
	"github.com/golang/protobuf/proto"
)

const (
	MarketsSummariesObjectNameV1   = "markets.pb"
	MarketsSummariesObjectNameV2   = "markets"
	MarketDetailObjectNameV1Format = "augur/markets/%s"

	MarketsSnapshotObjectNameV1 = "snapshot"
)

type Writer struct {
	Bucket           string
	GCloudStorageAPI *storage.Client
}

func (w *Writer) WriteMarketsSummary(summary *markets.MarketsSummary) error {
	protobuf, err := proto.Marshal(summary)
	if err != nil {
		return err
	}

	// Attempt to write all objects, accumulate errors
	errs := []error{}

	// V1
	if err := gcloud.WriteObject(w.GCloudStorageAPI, gcloud.WriteObjectParameters{
		Bucket:     w.Bucket,
		ObjectName: MarketsSummariesObjectNameV1,
		Content:    protobuf,
	}, func(wrtr *storage.Writer) {
		wrtr.ContentType = "application/octet-stream"
		wrtr.CacheControl = "public, max-age=15"
		wrtr.ACL = []storage.ACLRule{
			{storage.AllUsers, storage.RoleReader},
		}
	}); err != nil {
		errs = append(errs, err)
	}

	// V2
	gzipped := bytes.NewBuffer(nil)
	gzipWrtr := gzip.NewWriter(gzipped)
	if _, err := gzipWrtr.Write(protobuf); err != nil {
		return err
	}
	if err := gzipWrtr.Close(); err != nil {
		return err
	}
	if err := gcloud.WriteObject(w.GCloudStorageAPI, gcloud.WriteObjectParameters{
		Bucket:     w.Bucket,
		ObjectName: MarketsSummariesObjectNameV2,
		Content:    gzipped.Bytes(),
	}, func(wrtr *storage.Writer) {
		wrtr.ContentType = "application/octet-stream"
		wrtr.CacheControl = "public, max-age=15"
		wrtr.ContentEncoding = "gzip"
		wrtr.ACL = []storage.ACLRule{
			{storage.AllUsers, storage.RoleReader},
		}
	}); err != nil {
		errs = append(errs, err)
	}

	if len(errs) == 0 {
		return nil
	}

	// Aggregate errors and return
	resp := "Failed to upload markets summary: "
	for i, err := range errs {
		resp = fmt.Sprintf("%s || Error #%d: %s", resp, i, err.Error())
	}
	return fmt.Errorf(resp)
}

func (w *Writer) WriteMarketsSnapshot(snapshot *markets.MarketsSnapshot) error {
	protobuf, err := proto.Marshal(snapshot)
	if err != nil {
		return err
	}

	if err := gcloud.WriteObject(w.GCloudStorageAPI, gcloud.WriteObjectParameters{
		Bucket:     w.Bucket,
		ObjectName: MarketsSnapshotObjectNameV1,
		Content:    protobuf,
	}, func(wrtr *storage.Writer) {
		wrtr.ContentType = "application/octet-stream"
		wrtr.CacheControl = "public, max-age=15"
		wrtr.ACL = []storage.ACLRule{
			{storage.AllUsers, storage.RoleReader},
		}
	}); err != nil {
		return err
	}
	return nil
}

func (w *Writer) WriteMarketDetail(detail *markets.MarketDetail) error {
	protobuf, err := proto.Marshal(detail)
	if err != nil {
		return err
	}

	gzipped := bytes.NewBuffer(nil)
	gzipWrtr := gzip.NewWriter(gzipped)
	if _, err := gzipWrtr.Write(protobuf); err != nil {
		return err
	}
	if err := gzipWrtr.Close(); err != nil {
		return err
	}
	if err := gcloud.WriteObject(w.GCloudStorageAPI, gcloud.WriteObjectParameters{
		Bucket:     w.Bucket,
		ObjectName: fmt.Sprintf(MarketDetailObjectNameV1Format, strings.ToLower(detail.MarketId)),
		Content:    gzipped.Bytes(),
	}, func(wrtr *storage.Writer) {
		wrtr.ContentType = "application/octet-stream"
		wrtr.CacheControl = "public, max-age=15"
		wrtr.ContentEncoding = "gzip"
		wrtr.ACL = []storage.ACLRule{
			{storage.AllUsers, storage.RoleReader},
		}
	}); err != nil {
		return err
	}
	return nil
}
