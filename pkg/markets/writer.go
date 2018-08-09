package markets

import (
	"fmt"
	"strings"

	"github.com/stateshape/augur-analyzer/pkg/gcloud"
	"github.com/stateshape/augur-analyzer/pkg/proto/markets"

	"cloud.google.com/go/storage"
)

const (
	MarketsSummariesObjectNameV1   = "markets.pb"
	MarketsSummariesObjectNameV2   = "markets"
	MarketDetailObjectNameV1Format = "augur/markets/%s"

	MarketsSnapshotObjectNameV1 = "snapshot"
)

type Writer struct {
	Bucket         string
	ObjectUploader *gcloud.ObjectUploader
}

func (w *Writer) WriteMarketsSummary(summary *markets.MarketsSummary) error {
	return w.ObjectUploader.WriteObject(&gcloud.UploadObject{
		Msg:    summary,
		Bucket: w.Bucket,
		Object: MarketsSummariesObjectNameV2,
		IsGZIP: true,
		WriterModifier: func(wrtr *storage.Writer) {
			wrtr.ContentType = "application/octet-stream"
			wrtr.CacheControl = "public, max-age=15"
			wrtr.ContentEncoding = "gzip"
			wrtr.ACL = []storage.ACLRule{
				{storage.AllUsers, storage.RoleReader},
			}
		},
	})
}

func (w *Writer) WriteMarketsSnapshot(snapshot *markets.MarketsSnapshot) error {
	return w.ObjectUploader.WriteObject(&gcloud.UploadObject{
		Msg:    snapshot,
		Bucket: w.Bucket,
		Object: MarketsSnapshotObjectNameV1,
		IsGZIP: true,
		WriterModifier: func(wrtr *storage.Writer) {
			wrtr.ContentType = "application/octet-stream"
			wrtr.CacheControl = "public, max-age=15"
			wrtr.ContentEncoding = "gzip"
			wrtr.ACL = []storage.ACLRule{
				{storage.AllUsers, storage.RoleReader},
			}
		},
	})
}

func (w *Writer) WriteMarketDetail(object string, detail *markets.MarketDetailByMarketId) error {
	return w.ObjectUploader.WriteObject(&gcloud.UploadObject{
		Msg:    detail,
		Bucket: w.Bucket,
		Object: fmt.Sprintf(MarketDetailObjectNameV1Format, strings.ToLower(object)),
		IsGZIP: true,
		WriterModifier: func(wrtr *storage.Writer) {
			wrtr.ContentType = "application/octet-stream"
			wrtr.CacheControl = "public, max-age=15"
			wrtr.ContentEncoding = "gzip"
			wrtr.ACL = []storage.ACLRule{
				{storage.AllUsers, storage.RoleReader},
			}
		},
	})
}
