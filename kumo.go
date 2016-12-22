package main

import (
	"flag"
	"log"
	"os"

	"./kumo"
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

	config.Debug = *debug
	config.DebugFile = *debugFile
	config.Quiet = *quiet

	kumo := kumo.New(config)
	for _, filename := range files {
		if _, err := os.Stat(filename); err != nil {
			continue
		}
		if kumo.Get(filename) != nil {
			log.Printf("Error: %v", err)
		}
	}

	if !config.Quiet {
		println("⚑ All Done!")
	}
}
