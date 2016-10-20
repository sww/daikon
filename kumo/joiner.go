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
	DownloadPath   string
	Stop           chan bool
	Queue          chan *DecodedPart
	Logger         *dumblog.DumbLog
	TempPath       string
	mu             sync.Mutex
	segmentCount   map[string]int
	segmentTracker map[string]*fileTracker
	wait           *sync.WaitGroup
}

func InitJoiner(w *sync.WaitGroup) *Joiner {
	return &Joiner{
		DownloadPath:   "",
		Queue:          make(chan *DecodedPart),
		Stop:           make(chan bool, 1),
		segmentTracker: make(map[string]*fileTracker),
		segmentCount:   make(map[string]int),
		wait:           w,
	}
}

func (j *Joiner) SetSegmentCount(name string, count int) {
	j.mu.Lock()
	defer j.mu.Unlock()

	j.segmentCount[name] = count
}

func (j *Joiner) deleteSegmentCount(name string) {
	j.mu.Lock()
	defer j.mu.Unlock()

	delete(j.segmentCount, name)
}

func (j *Joiner) Run() {
	j.Logger.Printf("[JOINER] Joiner.Run()")
	for {
		select {
		case <-j.Stop:
			j.Logger.Printf("[JOINER] Joiner.Run() stopping")
			return
		case part := <-j.Queue:
			go func(part *DecodedPart) {
				defer j.Logger.Print("[JOINER] Done()")
				defer j.wait.Done()

				j.mu.Lock()
				tracker, exists := j.segmentTracker[part.Name]
				j.mu.Unlock()

				if !exists {
					j.Logger.Print("[JOINER] Part not in segmentTracker")

					tracker = new(fileTracker)
					tracker.current = 0
					j.mu.Lock()
					tracker.expected = j.segmentCount[part.SegmentName]
					j.segmentTracker[part.Name] = tracker
					j.mu.Unlock()
				}

				j.deleteSegmentCount(part.SegmentName)

				tracker.current++
				j.Logger.Print("[JOINER] tracker.current: ", tracker.current, ", tracker.expected: ", tracker.expected)

				if tracker.expected == tracker.current {
					j.Logger.Print("[JOINER] expected == current")
					j.wait.Add(1)
					j.Logger.Print("[JOINER] Add(1)")
					j.join(part.Name, tracker.expected)
					j.wait.Done()
					j.Logger.Print("[JOINER] Done()!")
				}
			}(part)
		default:
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func (j *Joiner) JoinAll() {
	if len(j.segmentTracker) == 0 {
		return
	}

	j.Logger.Print("[JOINER] JoinAll()")
	j.Logger.Printf("[JOINER] j.segmentTracker: %+v", j.segmentTracker)
	for k, tracker := range j.segmentTracker {
		if tracker.current != tracker.expected {
			j.join(k, tracker.expected)
		}
	}
}

func (j *Joiner) join(filename string, count int) {
	fullFilename := filepath.Join(j.DownloadPath, filename)
	fullFile, err := os.Create(fullFilename)
	defer fullFile.Close()

	defer func() {
		j.mu.Lock()
		delete(j.segmentTracker, filename)
		j.mu.Unlock()
	}()

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
