package daikon

import (
	"encoding/xml"
	"io"
	"io/ioutil"
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

type Segment struct {
	Bytes   int    `xml:"bytes,attr"`
	Number  int    `xml:"number,attr"`
	Segment string `xml:",chardata"`
	Group   string
}

func (n *NZB) Size() int {
	size := 0
	for i := 0; i < len(n.Files); i++ {
		files := n.Files[i]

		for j := 0; j < len(files.Segments); j++ {
			size += files.Segments[j].Bytes
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
