package config

import (
	"os"
	"strconv"
)

func IsTraceMode() bool {
	value, _ := strconv.ParseBool(os.Getenv(TraceEnvVar))
	return value
}

func IsDebugMode() bool {
	value, _ := strconv.ParseBool(os.Getenv(DebugEnvVar))
	return value
}
