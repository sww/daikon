package kumo

import (
	"encoding/xml"
	"io"
	"io/ioutil"
	"regexp"
)

type NZB struct {
	Files []File `xml:"file"`
}

type File struct {
	Poster   string    `xml:"poster,attr"`
	Subject  string    `xml:"subject,attr"`
	Date     int       `xml:"date,attr"`
	Groups   []string  `xml:"groups>group"`
	Segments []Segment `xml:"segments>segment"`
}

func (f *File) Extension() string {
	re := regexp.MustCompile("\\.\\w+")
	matches := re.FindAllString(f.Subject, -1)
	if len(matches) > 0 {
		return matches[len(matches)-1]
	}

	return ""
}

type Segment struct {
	Bytes   int64  `xml:"bytes,attr"`
	Number  int    `xml:"number,attr"`
	Segment string `xml:",chardata"`
	Group   string
}

func (n *NZB) Size() int64 {
	size := int64(0)
	for _, file := range n.Files {
		for _, segment := range file.Segments {
			size += segment.Bytes
		}
	}

	return size
}

func Parse(f io.Reader) (*NZB, error) {
	data, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}

	nzb := new(NZB)
	err = xml.Unmarshal(data, &nzb)
	if err != nil {
		return nil, err
	}

	return nzb, nil
}
