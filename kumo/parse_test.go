package kumo

import (
	"reflect"
	"strings"
	"testing"
)

func Test_Parse(t *testing.T) {
	content := strings.NewReader(`
<?xml version="1.0" encoding="utf-8" ?>
<!DOCTYPE nzb PUBLIC "-//newzBin//DTD NZB 1.0//EN" "http://www.newzbin.com/DTD/nzb/nzb-1.0.dtd">
    <nzb xmlns="http://www.newzbin.com/DTD/2003/nzb">
        <file poster="poster" date="123" subject="subject">
            <groups>
                <group>alt.group</group>
            </groups>
            <segments>
                <segment bytes="11" number="1">1@foo.com</segment>
            </segments>
        </file>
    </nzb>
`)
	nzb, err := Parse(content)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	want := NZB{
		Files: []File{
			{Poster: "poster", Subject: "subject", Date: 123, Groups: []string{"alt.group"}, Segments: []Segment{{Bytes: 11, Number: 1, Segment: "1@foo.com"}}}}}
	if !reflect.DeepEqual(nzb, &want) {
		t.Errorf("Returned %+v, want %+v", nzb, &want)
	}
}

func Test_Size(t *testing.T) {
	nzb := NZB{
		Files: []File{
			{Poster: "poster", Subject: "subject", Date: 123, Groups: []string{"alt.group"}, Segments: []Segment{{Bytes: 11, Number: 1, Segment: "1@foo.com"}}}}}

	want := int64(11)
	if want != nzb.Size() {
		t.Errorf("Returned %+v, want %+v", nzb, &want)
	}
}
