package kumo

import (
	"fmt"
	"hash/crc32"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sww/dumblog"
	"github.com/sww/yenc"
)

type Decode struct {
	JoinQueue chan *yenc.Part
	Logger    *dumblog.DumbLog
	Progress  *Progress
	Queue     chan string
	Stop      chan bool
	TempPath  string
	Wait      *sync.WaitGroup
}

func InitDecode(w *sync.WaitGroup) *Decode {
	return &Decode{
		Queue: make(chan string),
		Stop:  make(chan bool, 1),
		Wait:  w,
	}
}

func (d *Decode) Run() {
	for {
		select {
		case <-d.Stop:
			d.Logger.Print("[DECODE] Decode stopped")
			return
		case segment := <-d.Queue:
			d.Logger.Printf("[DECODE] Decode got %v", segment)
			go func() {
				part, err := d.decode(segment)
				if err != nil {
					d.Progress.isBroken = true
					d.Logger.Printf("[DECODE] d.Wait.Done() because of err:", err)
					d.Wait.Done()
				} else {
					d.JoinQueue <- part
				}
			}()
		default:
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func (d *Decode) decode(filename string) (*yenc.Part, error) {
	file, err := os.Open(filename)
	defer file.Close()

	if err != nil {
		return nil, err
	}

	part, err := yenc.Decode(file)
	if err != nil {
		return nil, err
	}

	defer os.Remove(filename)

	d.Logger.Print("[DECODE] Writing decoded file ", part.Name)

	partFilename := filepath.Join(d.TempPath, fmt.Sprintf("%v.%v", part.Name, part.BeginPart))
	ioutil.WriteFile(partFilename, part.Body, 0644)

	d.Logger.Printf("[DECODE] Adding %v to JoinQueue", part.Name)

	go func() {
		if !checksum(part.Body, part.CRC32) {
			d.Logger.Print("[DECODE] Checksums did not match")
			d.Progress.isBroken = true
		}
	}()

	return part, nil
}

func checksum(data []byte, expected string) bool {
	h := fmt.Sprintf("%08x", crc32.ChecksumIEEE(data))
	return strings.EqualFold(h, expected)
}
