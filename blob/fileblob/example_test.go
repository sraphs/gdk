package fileblob_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/sraphs/gdk/blob"
	"github.com/sraphs/gdk/blob/fileblob"
)

func ExampleOpenBucket() {
	// PRAGMA: This example is used on github.com/sraphs/gdk; PRAGMA comments adjust how it is shown and can be ignored.

	// The directory you pass to fileblob.OpenBucket must exist first.
	const myDir = "path/to/local/directory"
	if err := os.MkdirAll(myDir, 0777); err != nil {
		log.Fatal(err)
	}

	// Create a file-based bucket.
	bucket, err := fileblob.OpenBucket(myDir, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer bucket.Close()
}

func Example_openBucketFromURL() {
	// Create a temporary directory.
	dir, err := ioutil.TempDir("", "go-cloud-fileblob-example")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(dir)

	// On Unix, append the dir to "file://".
	// On Windows, convert "\" to "/" and add a leading "/":
	dirpath := filepath.ToSlash(dir)
	if os.PathSeparator != '/' && !strings.HasPrefix(dirpath, "/") {
		dirpath = "/" + dirpath
	}

	// blob.OpenBucket creates a *blob.Bucket from a URL.
	ctx := context.Background()
	b, err := blob.OpenBucket(ctx, "file://"+dirpath)
	if err != nil {
		log.Fatal(err)
	}
	defer b.Close()

	// Now we can use b to read or write files to the container.
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
