package kumo

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/sww/dumblog"
)

type Kumo struct {
	config   *Config
	download *Download
	decode   *Decode
	filter   *Filter
	join     *Joiner
	logger   *dumblog.DumbLog
	wait     *sync.WaitGroup
}

func New(config *Config) *Kumo {
	if _, err := os.Stat(config.Temp); err != nil {
		if os.Mkdir(config.Temp, 0775) != nil {
			log.Fatalf("Cannot make temp directory %v", config.Temp)
		}
	}

	var wait sync.WaitGroup

	download, err := InitDownload(config.Host, config.Username, config.Password, config.Port, config.Connections, config.SSL, &wait)
	if err != nil {
		log.Fatalf("Failed to InitDownload, with error: %v\n", err)
	}

	filter := NewFilter(config.Filters...)

	logger := dumblog.New(config.Debug)
	if config.DebugFile != "" {
		debugFile, err := os.Create(config.DebugFile)
		if err != nil {
			log.Fatalf("Error opening debug file: %v", err)
		}
		logger.SetOutput(debugFile)
		logger.Debug = true
	}

	decode := InitDecode(&wait)
	join := InitJoiner(&wait)

	download.DecodeQueue = decode.Queue
	decode.JoinQueue = join.Queue

	filter.Logger = logger
	download.Logger = logger
	decode.Logger = logger
	join.Logger = logger

	return &Kumo{
		config:   config,
		download: download,
		decode:   decode,
		join:     join,
		filter:   filter,
		logger:   logger,
		wait:     &wait,
	}
}

func (k *Kumo) Get(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	nzb, err := Parse(file)
	if err != nil {
		log.Fatalf("[KUMO] Failed to parse with error %v\n", err)
		return err
	}

	_, dirName := filepath.Split(strings.TrimSuffix(filename, filepath.Ext(filename)))
	tempPath := filepath.Join(k.config.Temp, dirName)
	downloadPath := filepath.Join(k.config.Download, dirName)

	k.download.TempPath = tempPath
	k.decode.TempPath = tempPath
	k.join.TempPath = tempPath
	k.join.DownloadPath = downloadPath

	k.logger.Printf("[KUMO] Creating temp path: '%v'", k.download.TempPath)
	os.Mkdir(k.download.TempPath, 0775)
	k.logger.Printf("[KUMO] Creating download path: '%v'", k.join.DownloadPath)
	os.Mkdir(k.join.DownloadPath, 0775)

	if k.filter.HasFilters() {
		nzb = k.filter.FilterNzb(nzb)
	}

	progress := NewProgress()

	for _, nzb = range k.filter.Split(nzb, ".par2") {
		progress.SetTotalSize(nzb.Size())

		k.download.Progress = progress
		k.decode.Progress = progress

		if !k.config.Quiet {
			go progress.Run()
		}

		k.get(nzb)

		k.logger.Printf("[KUMO] wait.Wait()")
		k.wait.Wait()
		k.join.JoinAll()

		progress.Done = true
		progress.Wait.Wait()

		if !progress.isBroken() {
			break
		} else {
			progress.reset()
			progress.prefix = PREFIX_PAR2
		}
	}

	os.RemoveAll(k.download.TempPath)

	return nil
}

func (k *Kumo) get(nzb *NZB) {
	k.logger.Printf("[KUMO] Size: %v", nzb.Size())

	go k.download.Run()
	go k.decode.Run()
	go k.join.Run()

	for _, nzbFile := range nzb.Files {
		numSegments := len(nzbFile.Segments)
		k.wait.Add(numSegments)
		k.logger.Print("[KUMO] Adding(", numSegments, ")")

		for _, segment := range nzbFile.Segments {
			k.logger.Print("[KUMO] Queuing ", segment)
			k.join.SetSegmentCount(segment.Segment, numSegments)
			segment.Group = nzbFile.Groups[0]
			k.download.Queue <- segment
		}
	}
}
