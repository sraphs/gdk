package memblob

import (
	"context"
	"net/http"
	"testing"

	"github.com/sraphs/gdk/blob"
	"github.com/sraphs/gdk/blob/driver"
	"github.com/sraphs/gdk/blob/drivertest"
)

type harness struct {
	prefix string
}

func newHarness(ctx context.Context, t *testing.T, prefix string) (drivertest.Harness, error) {
	return &harness{prefix: prefix}, nil
}

func (h *harness) HTTPClient() *http.Client {
	return nil
}

func (h *harness) MakeDriver(ctx context.Context) (driver.Bucket, error) {
	drv := openBucket(nil)
	if h.prefix == "" {
		return drv, nil
	}
	return driver.NewPrefixedBucket(drv, h.prefix), nil
}

func (h *harness) MakeDriverForNonexistentBucket(ctx context.Context) (driver.Bucket, error) {
	// Does not make sense for this driver.
	return nil, nil
}

func (h *harness) Close() {}

func TestConformance(t *testing.T) {
	newHarnessNoPrefix := func(ctx context.Context, t *testing.T) (drivertest.Harness, error) {
		return newHarness(ctx, t, "")
	}
	drivertest.RunConformanceTests(t, newHarnessNoPrefix, nil)
}

func TestConformanceWithPrefix(t *testing.T) {
	const prefix = "some/prefix/dir/"
	newHarnessWithPrefix := func(ctx context.Context, t *testing.T) (drivertest.Harness, error) {
		return newHarness(ctx, t, prefix)
	}
	drivertest.RunConformanceTests(t, newHarnessWithPrefix, nil)
}

func BenchmarkMemblob(b *testing.B) {
	drivertest.RunBenchmarks(b, OpenBucket(nil))
}

func TestOpenBucketFromURL(t *testing.T) {
	tests := []struct {
		URL     string
		WantErr bool
	}{
		// OK.
		{"mem://", false},
		// With prefix.
		{"mem://?prefix=foo/bar", false},
		// Invalid parameter.
		{"mem://?param=value", true},
	}

	ctx := context.Background()
	for _, test := range tests {
		b, err := blob.OpenBucket(ctx, test.URL)
		if b != nil {
			defer b.Close()
		}
		if (err != nil) != test.WantErr {
			t.Errorf("%s: got error %v, want error %v", test.URL, err, test.WantErr)
		}
	}
}
