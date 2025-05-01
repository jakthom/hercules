package herculespackage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/jakthom/hercules/pkg/db"
	"github.com/jakthom/hercules/pkg/metric"
	"github.com/jakthom/hercules/pkg/source"
	herculestypes "github.com/jakthom/hercules/pkg/types"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
	"sigs.k8s.io/yaml"
)

// Variables represents a map of variable values for package configuration.
type Variables map[string]interface{}

// Package concurrency constants.
const (
	// MaxConcurrentPackageInit is the maximum number of packages to initialize concurrently.
	MaxConcurrentPackageInit = 4
	// ErrorChannelSize is the size of the error channel buffer for package initialization.
	ErrorChannelSize = 3
)

// HTTP request timeout constants.
const (
	// DefaultHTTPTimeoutSeconds is the default timeout for HTTP requests to package endpoints.
	DefaultHTTPTimeoutSeconds = 30
)

// A hercules package consists of extensions, macros, sources, and metrics
// It can be downloaded from remote sources or shipped alongside hercules.

type Package struct {
	Name         herculestypes.PackageName  `json:"name"`
	Version      string                     `json:"version"`
	Variables    Variables                  `json:"variables"`
	Extensions   db.Extensions              `json:"extensions"`
	Macros       []db.Macro                 `json:"macros"`
	Sources      []source.Source            `json:"sources"`
	Metrics      metric.Definitions         `json:"metrics"`
	MetricPrefix herculestypes.MetricPrefix `json:"-"`
	Metadata     metric.Metadata            `json:"metadata"`
	// TODO -> Package-level secrets
}

// InitializePackagesWithConnection initializes multiple packages in parallel
// for better performance when starting up with many packages.
func InitializePackagesWithConnection(packages []Package, conn *sql.Conn) error {
	// Use errgroup to handle concurrent initialization with error handling.
	g := new(errgroup.Group)
	// Limit concurrency to avoid overwhelming the database connection.
	sem := make(chan struct{}, MaxConcurrentPackageInit)

	for i := range packages {
		pkg := &packages[i] // Capture package by reference in the loop.
		g.Go(func() error {
			// Semaphore to limit concurrency.
			sem <- struct{}{}
			defer func() { <-sem }()

			return pkg.InitializeWithConnection(conn)
		})
	}

	// Wait for all package initializations to complete
	return g.Wait()
}

func (p *Package) InitializeWithConnection(conn *sql.Conn) error {
	if len(p.Name) == 0 {
		log.Trace().Msg("empty package detected - skipping initialization")
		return nil
	}

	log.Info().Interface("package", p.Name).Msg("initializing " + string(p.Name) + " package")

	// Create a wait group for concurrent operations that don't return errors.
	var wg sync.WaitGroup

	// Error channel for collecting errors from goroutines.
	errChan := make(chan error, ErrorChannelSize)

	// Ensure extensions.
	wg.Add(1)
	go func() {
		defer wg.Done()
		db.EnsureExtensionsWithConnection(p.Extensions, conn)
	}()

	// Ensure macros.
	wg.Add(1)
	go func() {
		defer wg.Done()
		db.EnsureMacrosWithConnection(p.Macros, conn)
	}()

	// Ensure sources (this can return an error).
	go func() {
		err := source.InitializeSourcesWithConnection(p.Sources, conn)
		if err != nil {
			errChan <- fmt.Errorf("could not initialize sources for package %s: %w", p.Name, err)
		}
	}()

	// Inject metadata to all metrics.
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := p.Metrics.InjectMetadata(conn, p.Metadata); err != nil {
			errChan <- fmt.Errorf("could not inject metadata for package %s: %w", p.Name, err)
		}
	}()

	// Wait for all concurrent operations to finish.
	wg.Wait()

	// Check if any errors occurred during initialization.
	select {
	case err := <-errChan:
		log.Error().Err(err).Interface("package", p.Name).Msg("could not initialize package")
		return err
	default:
		// No error.
	}

	log.Info().Interface("package", p.Name).Msg(string(p.Name) + " package initialized")
	return nil
}

type PackageConfig struct {
	Package      string                     `json:"package"`
	Variables    Variables                  `json:"variables"`
	MetricPrefix herculestypes.MetricPrefix `json:"metricPrefix"`
}

func (p *PackageConfig) getFromFile() (Package, error) {
	log.Debug().Interface("package", p.Package).Msg("loading " + p.Package + " package from file")
	pkg := Package{}
	yamlFile, err := os.ReadFile(p.Package)
	if err != nil {
		log.Error().Err(err).Msg("could not get package from file " + p.Package)
	}
	err = yaml.Unmarshal(yamlFile, &pkg)
	pkg.MetricPrefix = p.MetricPrefix
	return pkg, err
}

func (p *PackageConfig) getFromEndpoint() (Package, error) {
	log.Debug().Interface("package", p.Package).Msg("loading " + p.Package + " package from endpoint")
	pkg := Package{}

	// Create a context with timeout for the HTTP request.
	ctx, cancel := context.WithTimeout(context.Background(), DefaultHTTPTimeoutSeconds*time.Second)
	defer cancel()

	// Create a request with context.
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.Package, nil)
	if err != nil {
		return pkg, fmt.Errorf("failed to create request: %w", err)
	}

	// Execute the request.
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Error().Err(err).Msg("could not get package from endpoint " + p.Package)
		return pkg, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error().Err(err).Msg("could not read response body from endpoint " + p.Package)
		return pkg, err
	}

	err = yaml.Unmarshal(body, &pkg)
	return pkg, err
}

func (p *PackageConfig) getFromObjectStorage() error {
	err := errors.New("object storage-backed packages are not yet supported")
	log.Fatal().Interface("package", p.Package).Err(err).Msg("failed to get package")
	return err
}

func (p *PackageConfig) GetPackage() (Package, error) {
	var pkg Package
	var err error
	switch p.Package[0:4] {
	case "http":
		pkg, err = p.getFromEndpoint()
	case "s3:/", "gcs:":
		err = p.getFromObjectStorage()
		return Package{}, err
	default:
		pkg, err = p.getFromFile()
	}
	if err != nil {
		log.Debug().Stack().Err(err).Msg("could not load package from location " + p.Package)
		return Package{}, err
	}
	pkg.Variables = p.Variables
	return pkg, nil
}
