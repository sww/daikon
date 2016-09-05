package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"./kumo"
	"github.com/sww/dumblog"
)

func main() {
	configName := flag.String("config", "config.json", "config file")
	debug := flag.Bool("debug", false, "show debug statements")
	quiet := flag.Bool("quiet", false, "hide the progress output")

	flag.Parse()

	configFile, err := os.Open(*configName)
	if err != nil {
		log.Fatalf("Error opening config file: %v\n", err)
	}

	config, err := kumo.GetConfig(configFile)
	if err != nil {
		log.Fatalf("Error reading config: %v\n", err)
	}

	files := flag.Args()
	if len(files) == 0 {
		log.Fatalf("[MAIN] No files specified")
	}

	logger := dumblog.DumbLog{Debug: *debug}
	downloadWait := new(sync.WaitGroup)

	// Make sure root directories are there.
	if _, err = os.Stat(config.Temp); err != nil {
		if os.Mkdir(config.Temp, 0775) != nil {
			log.Fatalf("Cannot make temp directory %v", config.Temp)
		}
	}

	download, err := kumo.InitDownload(config.Host, config.Username, config.Password, config.Port, config.Connections, downloadWait)
	if err != nil {
		log.Fatalf("Failed to InitDownload, with error: %v\n", err)
	}

	for _, filename := range files {
		if _, err := os.Stat(filename); err != nil {
			continue
		}

		decodeWait := new(sync.WaitGroup)
		decode := kumo.InitDecode(decodeWait)

		joinWait := new(sync.WaitGroup)
		join := kumo.InitJoiner(joinWait)

		download.DecodeQueue = decode.Queue
		decode.JoinQueue = join.Queue

		download.Logger = &logger
		decode.Logger = &logger
		join.Logger = &logger

		progress := kumo.InitProgress()
		download.Progress = progress
		decode.Progress = progress

		go download.Run()
		go decode.Run()
		go join.Run()

		file, err := os.Open(filename)
		if err != nil {
			continue
		}

		nzb, err := kumo.Parse(file)
		if err != nil {
			log.Fatalf("[MAIN] Failed to parse file \"%v\", with error %v\n", filename, err)
		}

		logger.Print("[MAIN] Files: ", len(nzb.Files))

		_, dirName := filepath.Split(strings.TrimSuffix(filename, filepath.Ext(filename)))

		download.TempPath = filepath.Join(config.Temp, dirName)
		decode.TempPath = filepath.Join(config.Temp, dirName)
		join.TempPath = filepath.Join(config.Temp, dirName)
		join.DownloadPath = filepath.Join(config.Download, dirName)

		if !*quiet {
			go progress.Run()
		}

		logger.Printf("[MAIN] Creating temp path: '%v'", download.TempPath)
		os.Mkdir(download.TempPath, 0775)
		logger.Printf("[MAIN] Creating download path: '%v'", join.DownloadPath)
		os.Mkdir(join.DownloadPath, 0775)

		for _, nzbFiles := range(nzb.Files) {
			logger.Printf("[MAIN] Size: %v", nzb.Size())

			progress.Total = nzb.Size()

			numSegments := len(nzbFiles.Segments)
			downloadWait.Add(numSegments)
			logger.Print("[MAIN] Adding(", numSegments, ")")

			for _, segment := range(nzbFiles.Segments) {
				logger.Print("[MAIN] Queuing ", segment)
				join.SegmentMap[segment.Segment] = numSegments
				segment.Group = nzbFiles.Groups[0]
				download.Queue <- segment
			}
		}

		logger.Print("[MAIN] downloadWait")
		downloadWait.Wait()
		logger.Print("[MAIN] decodeWait")
		decodeWait.Wait()

		join.JoinAll()
		logger.Print("[MAIN] joinWait")
		joinWait.Wait()

		progress.Done = true
		progress.Wait.Wait()

		os.RemoveAll(decode.TempPath)
	}

	if !*quiet {
		println("⚑ All Done!")
	}
}
