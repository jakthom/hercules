package config

import "os"

func IsTraceMode() bool {
	trace := os.Getenv(TraceEnvVar)
	return trace == "true" || trace == "1" || trace == "True"
}

func IsDebugMode() bool {
	debug := os.Getenv(DebugEnvVar)
	return debug == "true" || debug == "1" || debug == "True"
}
