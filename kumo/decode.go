package kumo

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/sww/dumblog"
	"github.com/sww/yenc"
)

type Decode struct {
	JoinQueue chan *yenc.Part
	Logger    *dumblog.DumbLog
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
			go d.decode(segment)
		default:
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func (d *Decode) decode(filename string) {
	file, err := os.Open(filename)
	defer file.Close()

	if err != nil {
		return
	}

	part, err := yenc.Decode(file)
	if err != nil {
		return
	}

	defer os.Remove(filename)

	if err != nil {
		d.Logger.Print("[DECODE] Done() because of err: ", err)
		return
	}

	d.Logger.Print("[DECODE] Writing decoded file ", part.Name)

	partFilename := filepath.Join(d.TempPath, fmt.Sprintf("%v.%v", part.Name, part.BeginPart))
	ioutil.WriteFile(partFilename, part.Body, 0644)

	d.Logger.Printf("[DECODE] Adding %v to JoinQueue", part.Name)

	d.JoinQueue <- part
}
