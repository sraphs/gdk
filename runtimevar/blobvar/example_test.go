package blobvar_test

import (
	"context"
	"fmt"
	"log"

	"github.com/sraphs/gdk/blob/memblob"
	"github.com/sraphs/gdk/runtimevar"
	"github.com/sraphs/gdk/runtimevar/blobvar"
)

func ExampleOpenVariable() {
	// Create a *blob.Bucket.
	// Here, we use an in-memory implementation and write a sample value.
	bucket := memblob.OpenBucket(nil)
	defer bucket.Close()
	ctx := context.Background()
	err := bucket.WriteAll(ctx, "cfg-variable-name", []byte("hello world"), nil)
	if err != nil {
		log.Fatal(err)
	}

	// Construct a *runtimevar.Variable that watches the blob.
	v, err := blobvar.OpenVariable(bucket, "cfg-variable-name", runtimevar.StringDecoder, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer v.Close()

	// We can now read the current value of the variable from v.
	snapshot, err := v.Latest(ctx)
	if err != nil {
		log.Fatal(err)
	}
	// runtimevar.Snapshot.Value is decoded to a string.
	fmt.Println(snapshot.Value.(string))

	// Output:
	// hello world
}

func Example_openVariableFromURL() {
	// PRAGMA: This example is used on github.com/sraphs/gdk; PRAGMA comments adjust how it is shown and can be ignored.
	// PRAGMA: On github.com/sraphs/gdk, add a blank import: _ "github.com/sraphs/gdk/runtimevar/blobvar"
	// PRAGMA: On github.com/sraphs/gdk, hide lines until the next blank line.
	ctx := context.Background()

	// runtimevar.OpenVariable creates a *runtimevar.Variable from a URL.
	// The default opener opens a blob.Bucket via a URL, based on the environment
	// variable BLOBVAR_BUCKET_URL.

	v, err := runtimevar.OpenVariable(ctx, "blob://myvar.txt?decoder=string")
	if err != nil {
		log.Fatal(err)
	}
	defer v.Close()
}
