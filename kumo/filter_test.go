package kumo

import (
	"testing"

	"github.com/sww/dumblog"
)

func TestFilter(t *testing.T) {
	filter := NewFilter("foo", "[a-c]+")
	filter.Logger = &dumblog.DumbLog{Debug: false}

	testStrings := []string{
		"foo",
		"abcabc",
	}

	for _, s := range testStrings {
		if filter.Filter(s) != true {
			t.Errorf("Filter() expected to return true for string %q", s)
		}
	}

	falseTestStrings := []string{
		"zzz",
		"def",
	}

	for _, s := range falseTestStrings {
		if filter.Filter(s) != false {
			t.Errorf("Filter() expected to return false for string %q", s)
		}
	}
}
