package main

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/jakthom/hercules/pkg/config"
	"github.com/jakthom/hercules/pkg/flock"
	herculespackage "github.com/jakthom/hercules/pkg/herculesPackage"
	"github.com/jakthom/hercules/pkg/metric"
	registry "github.com/jakthom/hercules/pkg/metricRegistry"
	"github.com/jakthom/hercules/pkg/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Hercules struct {
	config           config.Config
	db               *sql.DB
	packages         []herculespackage.Package
	conn             *sql.Conn
	metricRegistries []*registry.MetricRegistry
	debug            bool
	version          string // Added version field to the struct
}

func (d *Hercules) configure() {
	log.Debug().Msg("configuring Hercules")
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	debug := config.IsDebugMode()
	if debug {
		d.debug = true
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}
	trace := config.IsTraceMode()
	if trace {
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
	}

	// Load configuration and handle errors
	var err error
	d.config, err = config.GetConfig()
	if err != nil {
		log.Warn().Err(err).Msg("using default configuration due to error")
	} else if !debug && d.config.Debug {
		d.debug = true
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		log.Debug().Msg("debug mode enabled via config file")
	}
}

func (d *Hercules) initializeFlock() {
	log.Debug().Str("db", d.config.DB).Msg("initializing database")
	d.db, d.conn = flock.InitializeDB(d.config)
}

func (d *Hercules) loadPackages() {
	pkgs := []herculespackage.Package{}
	for _, pkgConfig := range d.config.Packages {
		pkg, err := pkgConfig.GetPackage()
		pkg.Metadata = metric.Metadata{
			PackageName: string(pkg.Name),
			Prefix:      pkgConfig.MetricPrefix,
			Labels:      d.config.InstanceLabels(),
		}
		if err != nil {
			log.Error().Err(err).Msg("could not get package")
		}
		pkgs = append(pkgs, pkg)
	}
	// Represent core configuration via a package
	pkgs = append(pkgs, herculespackage.Package{
		Name:       "core",
		Version:    "1.0.0",
		Extensions: d.config.Extensions,
		Macros:     d.config.Macros,
		Sources:    d.config.Sources,
		Metrics:    d.config.Metrics,
		Metadata: metric.Metadata{
			PackageName: "core",
			Labels:      d.config.InstanceLabels(),
		},
	})
	d.packages = pkgs
}

func (d *Hercules) initializePackages() {
	// Use our new parallel package initialization function
	err := herculespackage.InitializePackagesWithConnection(d.packages, d.conn)
	if err != nil {
		log.Error().Err(err).Msg("error initializing packages")
	}
}

func (d *Hercules) initializeRegistries() {
	// Register a registry for each package
	for _, pkg := range d.packages {
		if d.metricRegistries == nil {
			d.metricRegistries = []*registry.MetricRegistry{registry.NewMetricRegistry(pkg.Metrics)}
		} else {
			d.metricRegistries = append(d.metricRegistries, registry.NewMetricRegistry(pkg.Metrics))
		}
	}
}

func (d *Hercules) Initialize() {
	log.Debug().Msg("initializing Hercules")
	d.configure()
	d.initializeFlock()
	d.loadPackages()
	d.initializePackages()
	d.initializeRegistries()
	log.Debug().Interface("config", d.config).Msg("running with config")
}

func (d *Hercules) Run() {
	// Create server mux and configure routes
	mux := http.NewServeMux()
	prometheus.Unregister(collectors.NewGoCollector()) // Remove golang node defaults
	mux.Handle("/metrics", middleware.MetricsMiddleware(d.conn, d.metricRegistries, promhttp.Handler()))
	mux.Handle("/", http.RedirectHandler("/metrics", http.StatusSeeOther))

	// Server timeout constants
	const (
		readTimeoutSeconds     = 5
		writeTimeoutSeconds    = 10
		idleTimeoutSeconds     = 120
		shutdownTimeoutSeconds = 15
	)

	// Configure server with proper timeouts for better performance
	srv := &http.Server{
		Addr:         ":" + d.config.Port,
		Handler:      mux,
		ReadTimeout:  readTimeoutSeconds * time.Second,
		WriteTimeout: writeTimeoutSeconds * time.Second,
		IdleTimeout:  idleTimeoutSeconds * time.Second,
	}

	// Setup graceful shutdown with proper signal handling
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Start server in a goroutine
	serverErrors := make(chan error, 1)
	go func() {
		log.Info().Msg("hercules is running with version: " + d.version)
		serverErrors <- srv.ListenAndServe()
	}()

	// Block until we receive a signal or server error
	select {
	case err := <-serverErrors:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error().Err(err).Msg("server error")
		}
	case <-ctx.Done():
		log.Info().Msg("shutdown initiated")
	}

	// Create a timeout context for graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeoutSeconds*time.Second)
	defer shutdownCancel()

	// Use a WaitGroup to ensure proper cleanup of resources
	var wg sync.WaitGroup

	// Gracefully shut down the server
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Error().Err(err).Msg("server shutdown error")
		}
	}()

	// Close database connection in parallel
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Debug().Msg("closing database connection")
		if err := d.db.Close(); err != nil {
			log.Error().Err(err).Msg("error closing database connection")
		}
		if !d.debug {
			if err := os.Remove(d.config.DB); err != nil {
				log.Error().Err(err).Str("db", d.config.DB).Msg("error removing database file")
			}
		}
	}()

	// Wait for all cleanup operations to complete or timeout
	waitCh := make(chan struct{})
	go func() {
		wg.Wait()
		close(waitCh)
	}()

	select {
	case <-waitCh:
		log.Info().Msg("shutdown completed gracefully")
	case <-shutdownCtx.Done():
		log.Warn().Msg("shutdown timed out, forcing exit")
	}
}
