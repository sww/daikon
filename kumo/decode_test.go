package kumo

import (
	"reflect"
	"testing"
)

func Test_checksum(t *testing.T) {
	c := checksum([]byte("o"), "0F0F9344")
	want := true
	if !reflect.DeepEqual(c, want) {
		t.Errorf("Returned %+v, want %+v", c, want)
	}
}
