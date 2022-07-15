package memblob_test

import (
	"context"
	"fmt"
	"log"

	"github.com/sraphs/gdk/blob"
	"github.com/sraphs/gdk/blob/memblob"
)

func ExampleOpenBucket() {
	// PRAGMA: This example is used on github.com/sraphs/gdk; PRAGMA comments adjust how it is shown and can be ignored.
	// PRAGMA: On github.com/sraphs/gdk, hide lines until the next blank line.
	ctx := context.Background()

	// Create an in-memory bucket.
	bucket := memblob.OpenBucket(nil)
	defer bucket.Close()

	// Now we can use bucket to read or write files to the bucket.
	err := bucket.WriteAll(ctx, "my-key", []byte("hello world"), nil)
	if err != nil {
		log.Fatal(err)
	}
	data, err := bucket.ReadAll(ctx, "my-key")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(data))

	// Output:
	// hello world
}

func Example_openBucketFromURL() {
	// blob.OpenBucket creates a *blob.Bucket from a URL.
	b, err := blob.OpenBucket(context.Background(), "mem://")
	if err != nil {
		log.Fatal(err)
	}
	defer b.Close()

	// Now we can use b to read or write files to the container.
	ctx := context.Background()
	err = b.WriteAll(ctx, "my-key", []byte("hello world"), nil)
	if err != nil {
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
