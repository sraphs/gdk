package blob_test

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"

	"github.com/sraphs/gdk/blob"
	"github.com/sraphs/gdk/blob/memblob"
	"github.com/sraphs/gdk/gdkerr"
	"github.com/sraphs/gdk/internal/oc"
	"github.com/sraphs/gdk/internal/testing/octest"
)

func TestOpenCensus(t *testing.T) {
	ctx := context.Background()
	te := octest.NewTestExporter(blob.OpenCensusViews)
	defer te.Unregister()

	bytes := []byte("foo")
	b := memblob.OpenBucket(nil)
	defer b.Close()
	if err := b.WriteAll(ctx, "key", bytes, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := b.ReadAll(ctx, "key"); err != nil {
		t.Fatal(err)
	}
	if _, err := b.Attributes(ctx, "key"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := b.ListPage(ctx, blob.FirstPageToken, 3, nil); err != nil {
		t.Fatal(err)
	}
	if err := b.Delete(ctx, "key"); err != nil {
		t.Fatal(err)
	}
	if _, err := b.ReadAll(ctx, "noSuchKey"); err == nil {
		t.Fatal("got nil, want error")
	}

	const driver = "github.com/sraphs/gdk/blob/memblob"

	diff := octest.Diff(te.Spans(), te.Counts(), "github.com/sraphs/gdk/blob", driver, []octest.Call{
		{Method: "NewWriter", Code: gdkerr.OK},
		{Method: "NewRangeReader", Code: gdkerr.OK},
		{Method: "Attributes", Code: gdkerr.OK},
		{Method: "ListPage", Code: gdkerr.OK},
		{Method: "Delete", Code: gdkerr.OK},
		{Method: "NewRangeReader", Code: gdkerr.NotFound},
	})
	if diff != "" {
		t.Error(diff)
	}

	// Find and verify the bytes read/written metrics.
	var sawRead, sawWritten bool
	tags := []tag.Tag{{Key: oc.ProviderKey, Value: driver}}
	for !sawRead || !sawWritten {
		data := <-te.Stats
		switch data.View.Name {
		case "github.com/sraphs/gdk/blob/bytes_read":
			if sawRead {
				continue
			}
			sawRead = true
		case "github.com/sraphs/gdk/blob/bytes_written":
			if sawWritten {
				continue
			}
			sawWritten = true
		default:
			continue
		}
		if diff := cmp.Diff(data.Rows[0].Tags, tags, cmp.AllowUnexported(tag.Key{})); diff != "" {
			t.Errorf("tags for %s: %s", data.View.Name, diff)
			continue
		}
		sd, ok := data.Rows[0].Data.(*view.SumData)
		if !ok {
			t.Errorf("%s: data is %T, want SumData", data.View.Name, data.Rows[0].Data)
			continue
		}
		if got := int(sd.Value); got < len(bytes) {
			t.Errorf("%s: got %d, want at least %d", data.View.Name, got, len(bytes))
		}
	}
}
