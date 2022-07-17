package s3blob

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"

	"github.com/sraphs/gdk/blob"
	"github.com/sraphs/gdk/blob/driver"
	"github.com/sraphs/gdk/blob/drivertest"
	"github.com/sraphs/gdk/internal/testing/setup"
)

// These constants record the region & bucket used for the last --record.
// If you want to use --record mode,
// 1. Create a bucket in your AWS project from the S3 management console.
//
//	https://s3.console.aws.amazon.com/s3/home.
//
// 2. Update this constant to your bucket name.
// TODO(issue #300): Use Terraform to provision a bucket, and get the bucket
//
//	name from the Terraform output instead (saving a copy of it for replay).
const (
	bucketName = "go-cloud-testing"
	region     = "us-west-1"
)

type harness struct {
	client *s3.Client
	opts   *Options
	rt     http.RoundTripper
	closer func()
}

func newHarness(ctx context.Context, t *testing.T) (drivertest.Harness, error) {
	cfg, rt, done, _ := setup.NewAWSConfig(ctx, t, region)
	return &harness{client: s3.NewFromConfig(cfg), opts: nil, rt: rt, closer: done}, nil
}

func newHarnessUsingLegacyList(ctx context.Context, t *testing.T) (drivertest.Harness, error) {
	cfg, rt, done, _ := setup.NewAWSConfig(ctx, t, region)
	return &harness{client: s3.NewFromConfig(cfg), opts: &Options{UseLegacyList: true}, rt: rt, closer: done}, nil
}

func (h *harness) HTTPClient() *http.Client {
	return &http.Client{Transport: h.rt}
}

func (h *harness) MakeDriver(ctx context.Context) (driver.Bucket, error) {
	return openBucket(ctx, h.client, bucketName, h.opts)
}

func (h *harness) MakeDriverForNonexistentBucket(ctx context.Context) (driver.Bucket, error) {
	return openBucket(ctx, h.client, "bucket-does-not-exist", h.opts)
}

func (h *harness) Close() {
	h.closer()
}

func TestConformance(t *testing.T) {
	drivertest.RunConformanceTests(t, newHarness, []drivertest.AsTest{verifyContentLanguage{usingLegacyList: false}})
}

func TestConformanceUsingLegacyList(t *testing.T) {
	drivertest.RunConformanceTests(t, newHarnessUsingLegacyList, []drivertest.AsTest{verifyContentLanguage{usingLegacyList: true}})
}

func BenchmarkS3blob(b *testing.B) {

	bkt, err := OpenBucket(context.Background(), nil, bucketName, nil)
	if err != nil {
		b.Fatal(err)
	}
	drivertest.RunBenchmarks(b, bkt)
}

const language = "nl"

// verifyContentLanguage uses As to access the underlying GCS types and
// read/write the ContentLanguage field.
type verifyContentLanguage struct {
	usingLegacyList bool
}

func (verifyContentLanguage) Name() string {
	return "verify ContentLanguage can be written and read through As"
}

func (v verifyContentLanguage) BucketCheck(b *blob.Bucket) error {
	var client *s3.Client
	if !b.As(&client) {
		return errors.New("Bucket.As failed")
	}
	return nil
}

func (v verifyContentLanguage) ErrorCheck(b *blob.Bucket, err error) error {
	var e smithy.APIError
	if !b.ErrorAs(err, &e) {
		return errors.New("blob.ErrorAs failed")
	}
	return nil
}

func (v verifyContentLanguage) BeforeRead(as func(interface{}) bool) error {
	var req *s3.GetObjectInput
	if !as(&req) {
		return errors.New("BeforeRead As failed")
	}
	return nil
}

func (v verifyContentLanguage) BeforeWrite(as func(interface{}) bool) error {
	var req *s3.PutObjectInput
	if !as(&req) {
		return errors.New("Writer.As failed for PutObjectInput")
	}
	req.ContentLanguage = aws.String(language)
	var u *manager.Uploader
	if !as(&u) {
		return errors.New("Writer.As failed for Uploader")
	}
	return nil
}

func (v verifyContentLanguage) BeforeCopy(as func(interface{}) bool) error {
	var in *s3.CopyObjectInput
	if !as(&in) {
		return errors.New("BeforeCopy.As failed")
	}
	return nil
}

func (v verifyContentLanguage) BeforeList(as func(interface{}) bool) error {
	if v.usingLegacyList {
		var req *s3.ListObjectsInput
		if !as(&req) {
			return errors.New("List.As failed")
		}
	} else {
		var req *s3.ListObjectsV2Input
		if !as(&req) {
			return errors.New("List.As failed")
		}
	}
	return nil
}

func (v verifyContentLanguage) BeforeSign(as func(interface{}) bool) error {
	var (
		get *s3.GetObjectInput
		put *s3.PutObjectInput
		del *s3.DeleteObjectInput
	)
	if as(&get) || as(&put) || as(&del) {
		return nil
	}
	return errors.New("BeforeSign.As failed")
}

func (v verifyContentLanguage) AttributesCheck(attrs *blob.Attributes) error {
	var hoo s3.HeadObjectOutput
	if !attrs.As(&hoo) {
		return errors.New("Attributes.As returned false")
	}
	if got := *hoo.ContentLanguage; got != language {
		return fmt.Errorf("got %q want %q", got, language)
	}
	return nil
}

func (v verifyContentLanguage) ReaderCheck(r *blob.Reader) error {
	var goo s3.GetObjectOutput
	if !r.As(&goo) {
		return errors.New("Reader.As returned false")
	}
	if got := *goo.ContentLanguage; got != language {
		return fmt.Errorf("got %q want %q", got, language)
	}
	return nil
}

func (v verifyContentLanguage) ListObjectCheck(o *blob.ListObject) error {
	if o.IsDir {
		var commonPrefix types.CommonPrefix
		if !o.As(&commonPrefix) {
			return errors.New("ListObject.As for directory returned false")
		}
		return nil
	}
	var obj types.Object
	if !o.As(&obj) {
		return errors.New("ListObject.As for object returned false")
	}
	if obj.Key == nil || o.Key != *obj.Key {
		return errors.New("ListObject.As for object returned a different item")
	}
	return nil
}

func TestOpenBucket(t *testing.T) {
	tests := []struct {
		description string
		bucketName  string
		nilClient   bool
		want        string
		wantErr     bool
	}{
		{
			description: "empty bucket name results in error",
			wantErr:     true,
		},
		{
			description: "nil client results in error",
			bucketName:  "foo",
			nilClient:   true,
			wantErr:     true,
		},
		{
			description: "success",
			bucketName:  "foo",
			want:        "foo",
		},
	}

	ctx := context.Background()
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			var client *s3.Client
			if !test.nilClient {
				cfg, _, done, _ := setup.NewAWSConfig(ctx, t, region)
				defer done()
				client = s3.NewFromConfig(cfg)
			}

			// Create driver impl.
			drv, err := openBucket(ctx, client, test.bucketName, nil)
			if (err != nil) != test.wantErr {
				t.Errorf("got err %v want error %v", err, test.wantErr)
			}
			if err == nil && drv != nil && drv.name != test.want {
				t.Errorf("got %q want %q", drv.name, test.want)
			}

			// Create portable type.
			var b *blob.Bucket
			b, err = OpenBucket(ctx, client, test.bucketName, nil)
			if b != nil {
				defer b.Close()
			}
			if (err != nil) != test.wantErr {
				t.Errorf("got err %v want error %v", err, test.wantErr)
			}
		})
	}
}

func TestOpenBucketFromURL(t *testing.T) {
	tests := []struct {
		URL     string
		WantErr bool
	}{
		// OK.
		{"s3://my-bucket", false},
		// OK, setting region.
		{"s3://my-bucket?region=us-west1", false},
		// OK, setting profile.
		{"s3://my-bucket?profile=main", false},
		// OK, setting both profile and region.
		{"s3://my-bucket?profile=main&region=us-west-1", false},
		// OK, use V2.
		{"s3://my-bucket?awssdk=2", false},
		// Invalid parameter together with a valid one.
		{"s3://my-bucket?profile=main&param=value", true},
		// Invalid parameter.
		{"s3://my-bucket?param=value", true},
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
