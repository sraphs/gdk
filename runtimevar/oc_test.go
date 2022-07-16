package runtimevar_test

import (
	"context"
	"testing"

	"go.opencensus.io/stats/view"

	"github.com/sraphs/gdk/internal/oc"
	"github.com/sraphs/gdk/internal/testing/octest"
	"github.com/sraphs/gdk/runtimevar"
	"github.com/sraphs/gdk/runtimevar/constantvar"
)

func TestOpenCensus(t *testing.T) {
	ctx := context.Background()
	te := octest.NewTestExporter(runtimevar.OpenCensusViews)
	defer te.Unregister()

	v := constantvar.New(1)
	defer v.Close()
	if _, err := v.Watch(ctx); err != nil {
		t.Fatal(err)
	}
	cctx, cancel := context.WithCancel(ctx)
	cancel()
	_, _ = v.Watch(cctx)

	seen := false
	const driver = "github.com/sraphs/gdk/runtimevar/constantvar"
	for _, row := range te.Counts() {
		if _, ok := row.Data.(*view.CountData); !ok {
			continue
		}
		if row.Tags[0].Key == oc.ProviderKey && row.Tags[0].Value == driver {
			seen = true
			break
		}
	}
	if !seen {
		t.Errorf("did not see count row with provider=%s", driver)
	}
}
