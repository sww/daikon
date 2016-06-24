package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"./daikon"
	"github.com/sww/dumblog"
)

func main() {
	configName := flag.String("config", "config.json", "config file")
	debug := flag.Bool("debug", false, "show debug statements")

	flag.Parse()

	configFile, err := os.Open(*configName)
	if err != nil {
		log.Fatal("Error opening config file: %v\n", err)
	}

	config, err := daikon.GetConfig(configFile)
	if err != nil {
		log.Fatalf("Error reading config: %v\n", err)
	}

	files := flag.Args()
	if len(files) == 0 {
		log.Fatalf("[MAIN] No files specified")
	}

	logger := dumblog.DumbLog{Debug: *debug}
	wait := new(sync.WaitGroup)

	for i := 0; i < len(files); i++ {
		download, err := daikon.InitDownload(config.Host, config.Username, config.Password, config.Port, config.Connections, wait)
		if err != nil {
			log.Fatalf("Failed to InitDownload, with error: %v\n", err)
		}

		decode := daikon.InitDecode(wait)
		join := daikon.InitJoiner(wait)

		download.DecodeQueue = decode.Queue
		decode.JoinQueue = join.Queue

		download.Logger = &logger
		decode.Logger = &logger
		join.Logger = &logger

		progress := daikon.InitProgress()
		download.Progress = progress

		go download.Run()
		go decode.Run()
		go join.Run()

		filename := files[i]
		nf, err := os.Open(filename)
		if err != nil {
			continue
		}

		nzb, err := daikon.Parse(nf)

		if err != nil {
			log.Fatalf("[MAIN] Failed to parse file \"%v\", with error %v\n", filename, err)
		}

		logger.Print("[MAIN] Files: ", len(nzb.Files))

		_, dirName := filepath.Split(strings.TrimSuffix(filename, filepath.Ext(filename)))

		download.TempPath = filepath.Join(config.Temp, dirName)
		decode.TempPath = filepath.Join(config.Temp, dirName)
		join.TempPath = filepath.Join(config.Temp, dirName)
		join.DownloadPath = filepath.Join(config.Download, dirName)

		go progress.Run()

		logger.Print("[MAIN] Creating temp path: ", download.TempPath)
		os.Mkdir(download.TempPath, 0775)
		logger.Print("[MAIN] Creating download path: ", join.DownloadPath)
		os.Mkdir(join.DownloadPath, 0775)

		for j := 0; j < len(nzb.Files); j++ {
			logger.Printf("[MAIN] Size: %v", nzb.Size())

			progress.Total = nzb.Size()

			nzbFiles := nzb.Files[j]

			numSegments := len(nzbFiles.Segments)
			wait.Add(numSegments)
			logger.Print("[MAIN] Adding(", numSegments, ")")

			for k := 0; k < numSegments; k++ {
				s := nzbFiles.Segments[k]
				logger.Print("[MAIN] Queuing", s)
				s.Group = nzbFiles.Groups[0]
				download.Queue <- &s
			}
		}

		logger.Print("[MAIN] Waiting")
		wait.Wait()
		progress.Wait.Wait()

		os.RemoveAll(decode.TempPath)
	}

	println("⚑ All Done!")
}
