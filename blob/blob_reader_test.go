package blob_test

import (
	"context"
	"testing"
	"testing/iotest"

	"github.com/sraphs/gdk/blob/memblob"
)

// TestReader verifies that blob.Reader implements io package interfaces correctly.
func TestReader(t *testing.T) {
	const myKey = "testkey"

	bucket := memblob.OpenBucket(nil)
	defer bucket.Close()

	// Get some random data, of a large enough size to require multiple
	// reads/writes given our buffer size of 1024.
	data, err := randomData(1024*10 + 10)
	if err != nil {
		t.Fatal(err)
	}

	// Write the data to a key.
	ctx := context.Background()
	bucket.WriteAll(ctx, myKey, data, nil)

	// Create a blob.Reader.
	r, err := bucket.NewReader(ctx, myKey, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	if err := iotest.TestReader(r, data); err != nil {
		t.Error(err)
	}
}
