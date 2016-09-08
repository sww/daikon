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
	Map          map[string]*fileTracker
	Logger       *dumblog.DumbLog
	TempPath     string
	Wait         *sync.WaitGroup
	mu           sync.Mutex
	segmentCount map[string]int
}

func InitJoiner(w *sync.WaitGroup) *Joiner {
	return &Joiner{
		DownloadPath: "",
		Map:          make(map[string]*fileTracker),
		Queue:        make(chan *DecodedPart),
		Stop:         make(chan bool, 1),
		Wait:         w,
		segmentCount: make(map[string]int),
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
			go func() {
				defer j.Logger.Print("[JOINER] Done()")
				defer j.Wait.Done()

				tracker, exists := j.Map[part.Name]
				if !exists {
					j.Logger.Print("[JOINER] Part not in Map")

					tracker = new(fileTracker)
					tracker.current = 0
					tracker.expected = j.segmentCount[part.SegmentName]
					j.mu.Lock()
					j.Map[part.Name] = tracker
					j.mu.Unlock()
				}

				j.deleteSegmentCount(part.SegmentName)

				tracker.current++
				j.Logger.Print("[JOINER] tracker.current: ", tracker.current, ", tracker.expected: ", tracker.expected)

				if tracker.expected == tracker.current {
					j.Logger.Print("[JOINER] expected == current")
					j.Wait.Add(1)
					j.Logger.Print("[JOINER] Add(1)")
					j.join(part.Name, tracker.expected)
					j.Wait.Done()
					j.Logger.Print("[JOINER] Done()!")
				}
			}()
		default:
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func (j *Joiner) JoinAll() {
	if len(j.Map) == 0 {
		return
	}

	j.Logger.Print("[JOINER] JoinAll()")
	j.Logger.Printf("[JOINER] j.Map: %+v", j.Map)
	for k, tracker := range j.Map {
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
		delete(j.Map, filename)
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
