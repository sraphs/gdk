package main

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

func Test(t *testing.T) {
	got, fails, err := run(strings.NewReader(testOutput))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join("github.com/sraphs/gdk", "internal", "docstore", "internal", "fields", "TestUnexportedAnonymousNonStruct")
	want := fmt.Sprintf(`Failures (reporting up to 10):
  %s
ran 6; passed 3; failed 1; skipped 2 (in `, path)
	if !strings.HasPrefix(got, want) {
		t.Errorf("\ngot  %s\nwant %s", got, want)
	}
	if !fails {
		t.Error("wanted fails true, got false")
	}
}

const testOutput = `{"Time":"2019-05-09T16:39:56.83133991-04:00","Action":"run","Package":"github.com/sraphs/gdk/internal/docstore/internal/fields","Test":"TestFieldsNoTags"}
{"Time":"2019-05-09T16:39:56.831489481-04:00","Action":"output","Package":"github.com/sraphs/gdk/internal/docstore/internal/fields","Test":"TestFieldsNoTags","Output":"=== RUN   TestFieldsNoTags\n"}
{"Time":"2019-05-09T16:39:56.831517464-04:00","Action":"output","Package":"github.com/sraphs/gdk/internal/docstore/internal/fields","Test":"TestFieldsNoTags","Output":"--- PASS: TestFieldsNoTags (0.00s)\n"}
{"Time":"2019-05-09T16:39:56.831535431-04:00","Action":"pass","Package":"github.com/sraphs/gdk/internal/docstore/internal/fields","Test":"TestFieldsNoTags","Elapsed":0}
{"Time":"2019-05-09T16:39:56.831551807-04:00","Action":"run","Package":"github.com/sraphs/gdk/internal/docstore/internal/fields","Test":"TestAgainstJSONEncodingNoTags"}
{"Time":"2019-05-09T16:39:56.831561396-04:00","Action":"output","Package":"github.com/sraphs/gdk/internal/docstore/internal/fields","Test":"TestAgainstJSONEncodingNoTags","Output":"=== RUN   TestAgainstJSONEncodingNoTags\n"}
{"Time":"2019-05-09T16:39:56.831573783-04:00","Action":"output","Package":"github.com/sraphs/gdk/internal/docstore/internal/fields","Test":"TestAgainstJSONEncodingNoTags","Output":"--- PASS: TestAgainstJSONEncodingNoTags (0.00s)\n"}
{"Time":"2019-05-09T16:39:56.831584528-04:00","Action":"pass","Package":"github.com/sraphs/gdk/internal/docstore/internal/fields","Test":"TestAgainstJSONEncodingNoTags","Elapsed":0}
{"Time":"2019-05-09T16:39:56.844376487-04:00","Action":"output","Package":"github.com/sraphs/gdk/internal/docstore/drivertest","Output":"?   \tgithub.com/sraphs/gdk/internal/docstore/drivertest\t[no test files]\n"}
{"Time":"2019-05-09T16:39:56.844397339-04:00","Action":"skip","Package":"github.com/sraphs/gdk/internal/docstore/drivertest","Elapsed":0}
{"Time":"2019-05-09T16:39:56.831666898-04:00","Action":"output","Package":"github.com/sraphs/gdk/internal/docstore/internal/fields","Test":"TestFieldsWithTags","Output":"--- PASS: TestFieldsWithTags (0.00s)\n"}
{"Time":"2019-05-09T16:39:56.831677054-04:00","Action":"pass","Package":"github.com/sraphs/gdk/internal/docstore/internal/fields","Test":"TestFieldsWithTags","Elapsed":0}
{"Time":"2019-05-09T16:39:56.831729957-04:00","Action":"output","Package":"github.com/sraphs/gdk/internal/docstore/internal/fields","Test":"TestUnexportedAnonymousNonStruct","Output":"=== RUN   TestUnexportedAnonymousNonStruct\n"}
{"Time":"2019-05-09T16:39:56.831759258-04:00","Action":"fail","Package":"github.com/sraphs/gdk/internal/docstore/internal/fields","Test":"TestUnexportedAnonymousNonStruct","Elapsed":0}
{"Time":"2019-05-09T16:39:56.873905964-04:00","Action":"skip","Package":"github.com/sraphs/gdk/internal/docstore/memdocstore","Test":"TestConformance/TypeDrivenCodec","Elapsed":0}
`
