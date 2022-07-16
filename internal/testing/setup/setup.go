package setup

import (
	"context"
	"flag"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	awsv2 "github.com/aws/aws-sdk-go-v2/aws"
	awsv2config "github.com/aws/aws-sdk-go-v2/config"
	awsv2creds "github.com/aws/aws-sdk-go-v2/credentials"

	"github.com/google/go-replayers/httpreplay"
)

// Record is true iff the tests are being run in "record" mode.
var Record = flag.Bool("record", false, "whether to run tests against cloud resources and record the interactions")

// HasDockerTestEnvironment returns true when either:
// 1) Not on Github Actions.
// 2) On Github's Linux environment, where Docker is available.
func HasDockerTestEnvironment() bool {
	s := os.Getenv("RUNNER_OS")
	return s == "" || s == "Linux"
}

// NewRecordReplayClient creates a new http.Client for tests. This client's
// activity is being either recorded to files (when *Record is set) or replayed
// from files. rf is a modifier function that will be invoked with the address
// of the httpreplay.Recorder object used to obtain the client; this function
// can mutate the recorder to add service-specific header filters, for example.
// An initState is returned for tests that need a state to have deterministic
// results, for example, a seed to generate random sequences.
func NewRecordReplayClient(ctx context.Context, t *testing.T, rf func(r *httpreplay.Recorder)) (c *http.Client, cleanup func(), initState int64) {
	httpreplay.DebugHeaders()
	path := filepath.Join("testdata", t.Name()+".replay")
	if *Record {
		t.Logf("Recording into golden file %s", path)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		state := time.Now()
		b, _ := state.MarshalBinary()
		rec, err := httpreplay.NewRecorder(path, b)
		if err != nil {
			t.Fatal(err)
		}
		rf(rec)
		cleanup = func() {
			if err := rec.Close(); err != nil {
				t.Fatal(err)
			}
		}

		return rec.Client(), cleanup, state.UnixNano()
	}
	t.Logf("Replaying from golden file %s", path)
	rep, err := httpreplay.NewReplayer(path)
	if err != nil {
		t.Fatal(err)
	}
	recState := new(time.Time)
	if err := recState.UnmarshalBinary(rep.Initial()); err != nil {
		t.Fatal(err)
	}
	return rep.Client(), func() { rep.Close() }, recState.UnixNano()
}

// NewAWSv2Config creates a new aws.Config for testing against AWS.
// If the test is in --record mode, the test will call out to AWS, and the
// results are recorded in a replay file.
// Otherwise, the session reads a replay file and runs the test as a replay,
// which never makes an outgoing HTTP call and uses fake credentials.
// An initState is returned for tests that need a state to have deterministic
// results, for example, a seed to generate random sequences.
func NewAWSv2Config(ctx context.Context, t *testing.T, region string) (cfg awsv2.Config, rt http.RoundTripper, cleanup func(), initState int64) {
	client, cleanup, state := NewRecordReplayClient(ctx, t, func(r *httpreplay.Recorder) {
		r.RemoveQueryParams("X-Amz-Credential", "X-Amz-Signature", "X-Amz-Security-Token")
		r.RemoveRequestHeaders("Authorization", "Duration", "X-Amz-Security-Token")
		r.ClearHeaders("Amz-Sdk-Invocation-Id")
		r.ClearHeaders("X-Amz-Date")
		r.ClearQueryParams("X-Amz-Date")
		r.ClearHeaders("User-Agent") // AWS includes the Go version
	})
	cfg, err := awsV2Config(ctx, region, client)
	if err != nil {
		t.Fatal(err)
	}
	return cfg, client.Transport, cleanup, state
}

func awsV2Config(ctx context.Context, region string, client *http.Client) (awsv2.Config, error) {
	// Provide fake creds if running in replay mode.
	var creds awsv2.CredentialsProvider
	if !*Record {
		creds = awsv2creds.NewStaticCredentialsProvider("FAKE_KEY", "FAKE_SECRET", "FAKE_SESSION")
	}
	return awsv2config.LoadDefaultConfig(
		ctx,
		awsv2config.WithHTTPClient(client),
		awsv2config.WithRegion(region),
		awsv2config.WithCredentialsProvider(creds),
		awsv2config.WithRetryer(func() awsv2.Retryer { return awsv2.NopRetryer{} }),
	)
}
