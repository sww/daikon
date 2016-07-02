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

type Tracker struct {
	expected int
	current  int
}

type Joiner struct {
	DownloadPath string
	Stop         chan bool
	Queue        chan *yenc.Part
	Map          map[string]*Tracker
	Logger       *dumblog.DumbLog
	TempPath     string
	Wait         *sync.WaitGroup
}

func InitJoiner(w *sync.WaitGroup) *Joiner {
	return &Joiner{
		DownloadPath: "",
		Map:          make(map[string]*Tracker),
		Queue:        make(chan *yenc.Part),
		Stop:         make(chan bool, 1),
		Wait:         w,
	}
}

func (j *Joiner) Run() {
	j.Logger.Printf("[JOINER] Joiner.Run()")
	if _, err := os.Stat(j.DownloadPath); os.IsNotExist(err) {
		j.Logger.Print("[JOINER] Created directory", j.DownloadPath)
		os.Mkdir(j.DownloadPath, 0775)
	}

	for {
		select {
		case <-j.Stop:
			j.Logger.Printf("[JOINER] Joiner.Run() stopping")
			return
		case part := <-j.Queue:
			tracker, exists := j.Map[part.Name]
			if !exists {
				j.Logger.Print("[JOINER] Part not in Map")

				tracker = new(Tracker)
				tracker.current = 0
				j.Map[part.Name] = tracker
			}

			j.Logger.Print("[JOINER] Part.BeginPart, Part.EndPart: ", part.BeginSize == part.EndSize)
			if part.BeginSize == part.PartEnd {
				j.Logger.Print("[JOINER] Part.BeginSize == Part.EndSize")
				tracker.expected = part.BeginPart
			} else {
				j.Logger.Print("[JOINER] Part.BeginPart != Part.EndPart")
			}

			tracker.current++
			j.Logger.Print("[JOINER] tracker.current: ", tracker.current)
			j.Logger.Print("[JOINER] tracker.expected: ", tracker.expected)
			if tracker.expected == tracker.current {
				j.Logger.Print("[JOINER] expected == current")
				go j.join(part.Name, tracker.expected)
			}
		default:
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func (j *Joiner) join(filename string, count int) {
	fullFilename := filepath.Join(j.DownloadPath, filename)
	fullFile, err := os.Create(fullFilename)

	defer func() {
		j.Logger.Printf("[JOINER] Calling Done() %v times", count)
		for i := 0; i < count; i++ {
			j.Wait.Done()
		}
	}()
	defer fullFile.Close()

	if err != nil {
		j.Logger.Print("[JOINER] Create fullFile err: ", err)
		return
	}

	// TODO: Fix broken file?

	j.Logger.Print("[JOINER] Joiner.Join(", filename, ", ", count, ")")

	if err != nil {
		j.Logger.Print("[JOINER] error opening ", filename, ": ", err)
		// TODO: Do something.
		return
	}

	bytesWritten := 0

	for i := 1; i < count+1; i++ {
		partFilename := filepath.Join(j.TempPath, fmt.Sprintf("%v.%v", filename, i))
		j.Logger.Print("[JOINER] Joining decoded part ", partFilename)
		file, err := os.Open(partFilename)
		if err != nil {
			// Probably a missing segment, but continue...
			j.Logger.Print(partFilename, "does not exist!")
			continue
		}

		defer os.Remove(partFilename)

		data, err := ioutil.ReadAll(file)
		if err != nil {
			j.Logger.Print("[JOINER] got err joining file: ", err)
			// Probably a broken file, but continue...
			continue
		}

		fullFile.Write(data)
		bytesWritten += len(data)
	}

	j.Logger.Print("[JOINER] Done joining file ", filename)
	j.Logger.Print("[JOINER] Wrote ", bytesWritten, " bytes")
}
