package s3blob_test

import (
	"context"
	"log"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/sraphs/gdk/blob"
	"github.com/sraphs/gdk/blob/s3blob"
)

func ExampleOpenBucket() {
	// PRAGMA: This example is used on github.com/sraphs/gdk; PRAGMA comments adjust how it is shown and can be ignored.
	// PRAGMA: On github.com/sraphs/gdk, hide lines until the next blank line.

	// Establish a AWS V2 Config.
	// See https://aws.github.io/aws-sdk-go-v2/docs/configuring-sdk/ for more info.
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatal(err)
	}

	// Create a *blob.Bucket.
	client := s3.NewFromConfig(cfg)
	bucket, err := s3blob.OpenBucket(ctx, client, "my-bucket", nil)
	if err != nil {
		log.Fatal(err)
	}
	defer bucket.Close()
}

func Example_openBucketFromURL() {
	// PRAGMA: This example is used on github.com/sraphs/gdk; PRAGMA comments adjust how it is shown and can be ignored.
	// PRAGMA: On github.com/sraphs/gdk, add a blank import: _ "github.com/sraphs/gdk/blob/s3blob"
	// PRAGMA: On github.com/sraphs/gdk, hide lines until the next blank line.
	ctx := context.Background()

	// blob.OpenBucket creates a *blob.Bucket from a URL.
	bucket, err := blob.OpenBucket(ctx, "s3://my-bucket?region=us-west-1")
	if err != nil {
		log.Fatal(err)
	}
	defer bucket.Close()

	// Forcing AWS SDK V2.
	bucket, err = blob.OpenBucket(ctx, "s3://my-bucket?region=us-west-1&awssdk=2")
	if err != nil {
		log.Fatal(err)
	}
	defer bucket.Close()
}
