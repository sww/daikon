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
	debugFile := flag.String("debugFile", "", "write debug statments to debugFile")
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

	logger := dumblog.New(*debug)
	if *debugFile != "" {
		df, err := os.Create(*debugFile)
		if err != nil {
			log.Fatalf("Error opening debug file: %v", err)
		}
		logger.SetOutput(df)
		logger.Debug = true
	}
	wait := new(sync.WaitGroup)

	// Make sure root directories are there.
	if _, err = os.Stat(config.Temp); err != nil {
		if os.Mkdir(config.Temp, 0775) != nil {
			log.Fatalf("Cannot make temp directory %v", config.Temp)
		}
	}

	download, err := kumo.InitDownload(config.Host, config.Username, config.Password, config.Port, config.Connections, config.SSL, wait)
	if err != nil {
		log.Fatalf("Failed to InitDownload, with error: %v\n", err)
	}

	filter := kumo.NewFilter(config.Filters...)

	for _, filename := range files {
		if _, err := os.Stat(filename); err != nil {
			continue
		}

		decode := kumo.InitDecode(wait)
		join := kumo.InitJoiner(wait)

		download.DecodeQueue = decode.Queue
		decode.JoinQueue = join.Queue

		download.Logger = logger
		decode.Logger = logger
		join.Logger = logger
		filter.Logger = logger

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

		if filter.HasFilters() {
			nzb = filter.FilterNzb(nzb)
		}

		for _, nzbFiles := range nzb.Files {
			logger.Printf("[MAIN] Size: %v", nzb.Size())

			progress.SetTotalSize(nzb.Size())

			numSegments := len(nzbFiles.Segments)
			wait.Add(numSegments)
			logger.Print("[MAIN] Adding(", numSegments, ")")

			for _, segment := range nzbFiles.Segments {
				logger.Print("[MAIN] Queuing ", segment)
				join.SetSegmentCount(segment.Segment, numSegments)
				segment.Group = nzbFiles.Groups[0]
				download.Queue <- segment
			}
		}

		logger.Printf("[MAIN] wait.Wait()")
		wait.Wait()
		join.JoinAll()

		progress.Done = true
		progress.Wait.Wait()

		os.RemoveAll(decode.TempPath)
	}

	if !*quiet {
		println("⚑ All Done!")
	}
}
