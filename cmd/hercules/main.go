package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	herculesruntime "github.com/jakthom/hercules/internal/runtime"
	"github.com/jakthom/hercules/pkg/config"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var version = "dev"

const shutdownTimeout = 15 * time.Second

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout io.Writer, stderr io.Writer) int {
	command, commandArgs := parseCommand(args)
	switch command {
	case "version":
		_, _ = fmt.Fprintln(stdout, version)
		return 0
	case "help":
		printUsage(stdout)
		return 0
	case "package-digest":
		return runPackageDigest(commandArgs, stdout, stderr)
	}

	configPath, ok := parseConfigPath(command, commandArgs, stderr)
	if !ok {
		return 2
	}
	switch command {
	case "check-config":
		return runStaticCheck(configPath, stdout, stderr)
	case "check":
		return runFullCheck(configPath, stdout, stderr)
	case "print-effective-config":
		return printEffectiveConfig(configPath, stdout, stderr)
	case "run":
		return runServer(configPath)
	default:
		_, _ = fmt.Fprintf(stderr, "unknown command %q\n", command)
		printUsage(stderr)
		return 2
	}
}

func parseCommand(args []string) (string, []string) {
	if len(args) == 0 || len(args[0]) == 0 || args[0][0] == '-' {
		return "run", args
	}
	return args[0], args[1:]
}

func parseConfigPath(command string, args []string, stderr io.Writer) (string, bool) {
	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	flags.SetOutput(stderr)
	configPath := flags.String("config", config.DefaultHerculesConfigPath, "path to the Hercules configuration file")
	if err := flags.Parse(args); err != nil {
		return "", false
	}
	if flags.NArg() != 0 {
		_, _ = fmt.Fprintf(stderr, "%s does not accept positional arguments\n", command)
		return "", false
	}
	return *configPath, true
}

func runStaticCheck(configPath string, stdout io.Writer, stderr io.Writer) int {
	err := herculesruntime.Check(context.Background(), herculesruntime.Options{
		ConfigPath: configPath,
		Version:    version,
	})
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	_, _ = fmt.Fprintln(stdout, "configuration and packages are valid")
	return 0
}

func runFullCheck(configPath string, stdout io.Writer, stderr io.Writer) int {
	configureLogging(configPath)
	// Build applies the configured package-fetch and collection timeouts. Do not
	// impose the shorter shutdown deadline on valid, deliberately long queries.
	createdRuntime, err := herculesruntime.Build(context.Background(), herculesruntime.Options{
		ConfigPath: configPath,
		Version:    version,
	})
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	shutdownContext, cancelShutdown := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancelShutdown()
	if err := createdRuntime.Close(shutdownContext); err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	_, _ = fmt.Fprintln(stdout, "configuration, packages, sources, and metrics are valid")
	return 0
}

func printEffectiveConfig(configPath string, stdout io.Writer, stderr io.Writer) int {
	loaded, err := config.Load(configPath)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	for index := range loaded.Config.Packages {
		for name := range loaded.Config.Packages[index].Variables {
			loaded.Config.Packages[index].Variables[name] = "<redacted>"
		}
	}
	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(loaded.Config); err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

func runPackageDigest(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) != 1 {
		_, _ = fmt.Fprintln(stderr, "usage: hercules package-digest PATH")
		return 2
	}
	// #nosec G304 -- this command intentionally hashes the exact local path selected by the operator.
	payload, err := os.ReadFile(args[0])
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	_, _ = fmt.Fprintf(stdout, "sha256:%x\n", sha256.Sum256(payload))
	return 0
}

func runServer(configPath string) int {
	configureLogging(configPath)
	rootContext, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	createdRuntime, err := herculesruntime.Build(rootContext, herculesruntime.Options{
		ConfigPath: configPath,
		Version:    version,
	})
	if err != nil {
		log.Error().Err(err).Msg("Hercules startup failed")
		return 1
	}
	runError := createdRuntime.Run(rootContext)
	shutdownContext, cancelShutdown := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancelShutdown()
	closeError := createdRuntime.Close(shutdownContext)
	if runError != nil || closeError != nil {
		log.Error().Err(errors.Join(runError, closeError)).Msg("Hercules stopped with an error")
		return 1
	}
	return 0
}

func configureLogging(configPath string) {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if config.IsDebugMode() {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}
	if config.IsTraceMode() {
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
	}
	if loaded, err := config.Load(configPath); err == nil && loaded.Config.Debug && !config.IsTraceMode() {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}
}

func printUsage(writer io.Writer) {
	_, _ = fmt.Fprintln(writer, `Hercules turns DuckDB SQL queries into immutable Prometheus metric snapshots.

Usage:
  hercules [run] [--config PATH]
  hercules check-config [--config PATH]
  hercules check [--config PATH]
  hercules print-effective-config [--config PATH]
  hercules package-digest PATH
  hercules version
  hercules help`)
}
