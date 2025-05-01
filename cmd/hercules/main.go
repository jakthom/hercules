package main

import (
	"flag"
	"os"
)

// version is set during build.
var version string

func main() {
	// Parse command line flags
	configPath := flag.String("config", "", "Path to the configuration file")
	flag.Parse()

	// Set environment variable for config path if specified via flag
	if *configPath != "" {
		if err := os.Setenv("HERCULES_CONFIG_PATH", *configPath); err != nil {
			panic("Failed to set HERCULES_CONFIG_PATH environment variable: " + err.Error())
		}
	}

	app := Hercules{
		version: version,
	}
	app.Initialize()
	app.Run()
}
