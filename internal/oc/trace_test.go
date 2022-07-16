package oc

import (
	"regexp"
	"testing"
)

type testDriver struct{}

func TestProviderName(t *testing.T) {
	for _, test := range []struct {
		in   interface{}
		want string
	}{
		{nil, ""},
		{testDriver{}, "github.com/sraphs/gdk/internal/oc"},
		{&testDriver{}, "github.com/sraphs/gdk/internal/oc"},
		{regexp.Regexp{}, "regexp"},
	} {
		got := ProviderName(test.in)
		if got != test.want {
			t.Errorf("%v: got %q, want %q", test.in, got, test.want)
		}
	}
}
