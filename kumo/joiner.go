package kumo

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/sww/dumblog"
)

type fileTracker struct {
	expected int
	current  int
}

type Joiner struct {
	DownloadPath string
	Stop         chan bool
	Queue        chan *DecodedPart
	SegmentMap   map[string]int
	Map          map[string]*fileTracker
	Logger       *dumblog.DumbLog
	TempPath     string
	Wait         *sync.WaitGroup
}

func InitJoiner(w *sync.WaitGroup) *Joiner {
	return &Joiner{
		DownloadPath: "",
		SegmentMap:   make(map[string]int),
		Map:          make(map[string]*fileTracker),
		Queue:        make(chan *DecodedPart),
		Stop:         make(chan bool, 1),
		Wait:         w,
	}
}

func (j *Joiner) Run() {
	j.Logger.Printf("[JOINER] Joiner.Run()")
	for {
		select {
		case <-j.Stop:
			j.Logger.Printf("[JOINER] Joiner.Run() stopping")
			return
		case part := <-j.Queue:
			j.Wait.Add(1)
			tracker, exists := j.Map[part.Name]
			if !exists {
				j.Logger.Print("[JOINER] Part not in Map")

				tracker = new(fileTracker)
				tracker.current = 0
				tracker.expected = j.SegmentMap[part.SegmentName]
				j.Map[part.Name] = tracker
			}

			delete(j.SegmentMap, part.SegmentName)

			tracker.current++
			j.Logger.Print("[JOINER] tracker.current: ", tracker.current, ", tracker.expected: ", tracker.expected)

			if tracker.expected == tracker.current {
				j.Logger.Print("[JOINER] expected == current")
				go j.join(part.Name, tracker.expected)
			}
		default:
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func (j *Joiner) JoinAll() {
	j.Logger.Print("[JOINER] JoinAll()")
	j.Logger.Printf("[JOINER] j.Map: %+v", j.Map)
	for k, tracker := range j.Map {
		if tracker.current != tracker.expected {
			// If a segment is broken, we call Done() before we get to join,
			// so add to the wait group the number of broken segments so that
			// we Done() calls don't go negative.
			j.Wait.Add(tracker.expected - tracker.current)
			j.join(k, tracker.expected)
		}
	}
}

func (j *Joiner) join(filename string, count int) {
	fullFilename := filepath.Join(j.DownloadPath, filename)
	fullFile, err := os.Create(fullFilename)
	defer fullFile.Close()

	defer func() {
		j.Logger.Printf("[JOINER] Calling Done() %v times", count)
		for i := 0; i < count; i++ {
			j.Wait.Done()
		}
	}()
	defer delete(j.Map, filename)

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
			j.Logger.Print("[JOINER] ", partFilename, " does not exist!")
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
