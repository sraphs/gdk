package blob_test

import (
	"context"
	"fmt"
	"log"

	"github.com/sraphs/gdk/blob"
	_ "github.com/sraphs/gdk/blob/memblob"
)

func Example_openFromURL() {
	ctx := context.Background()

	// Connect to a bucket using a URL.
	// This example uses "memblob", the in-memory implementation.
	// We need to add a blank import line to register the memblob driver's
	// URLOpener, which implements blob.BucketURLOpener:
	// import _ "github.com/sraphs/gdk/blob/memblob"
	// memblob registers for the "mem" scheme.
	// All blob.OpenBucket URLs also work with "blob+" or "blob+bucket+" prefixes,
	// e.g., "blob+mem://" or "blob+bucket+mem://".
	b, err := blob.OpenBucket(ctx, "mem://")
	if err != nil {
		log.Fatal(err)
	}
	defer b.Close()

	// Now we can use b to read or write to blobs in the bucket.
	if err := b.WriteAll(ctx, "my-key", []byte("hello world"), nil); err != nil {
		log.Fatal(err)
	}
	data, err := b.ReadAll(ctx, "my-key")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(data))
	// Output:
	// hello world
}

func Example_openFromURLWithPrefix() {
	// PRAGMA: This example is used on github.com/sraphs/gdk; PRAGMA comments adjust how it is shown and can be ignored.
	// PRAGMA: On github.com/sraphs/gdk, hide lines until the next blank line.
	ctx := context.Background()

	// Connect to a bucket using a URL, using the "prefix" query parameter to
	// target a subfolder in the bucket.
	// The prefix should end with "/", so that the resulting bucket operates
	// in a subfolder.
	b, err := blob.OpenBucket(ctx, "mem://?prefix=a/subfolder/")
	if err != nil {
		log.Fatal(err)
	}
	defer b.Close()

	// Bucket operations on <key> will be translated to "a/subfolder/<key>".
}

func Example_openFromURLWithSingleKey() {
	// PRAGMA: This example is used on github.com/sraphs/gdk; PRAGMA comments adjust how it is shown and can be ignored.
	// PRAGMA: On github.com/sraphs/gdk, hide lines until the next blank line.
	ctx := context.Background()

	// Connect to a bucket using a URL, using the "key" query parameter to
	// make the bucket always reference that key.
	b, err := blob.OpenBucket(ctx, "mem://?key=foo.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer b.Close()

	// Bucket operations will ignore the passed-in key and always reference foo.txt.
}
