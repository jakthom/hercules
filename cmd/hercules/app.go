package main

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dbecorp/hercules/pkg/config"
	"github.com/dbecorp/hercules/pkg/flock"
	herculespackage "github.com/dbecorp/hercules/pkg/herculesPackage"
	metrics "github.com/dbecorp/hercules/pkg/metrics"
	"github.com/dbecorp/hercules/pkg/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var VERSION string

type Hercules struct {
	config         config.Config
	db             *sql.DB
	packages       []herculespackage.Package
	conn           *sql.Conn
	metricRegistry *metrics.MetricRegistry
}

func (d *Hercules) configure() {
	log.Debug().Msg("configuring Hercules")
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	debug := os.Getenv(config.DEBUG)
	if debug != "" && (debug == "true" || debug == "1" || debug == "True") {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}
	trace := os.Getenv(config.TRACE)
	if trace != "" && (trace == "true" || trace == "1" || trace == "True") {
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
	}
	d.config, _ = config.GetConfig()
}

func (d *Hercules) initializeFlock() {
	d.db, d.conn = flock.InitializeDB(d.config)
}

func (d *Hercules) loadPackages() {
	pkgs := []herculespackage.Package{}
	for _, pkgConfig := range d.config.Packages {
		pkg, err := pkgConfig.GetPackage()
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
	})
	d.packages = pkgs

}

func (d *Hercules) initializePackages() {
	for _, p := range d.packages {
		p.InitializeWithConnection(d.conn)
	}
}

func (d *Hercules) initializeRegistry() {
	d.metricRegistry = metrics.NewMetricRegistry(d.config.Metrics, d.config.InstanceLabels())
}

func (d *Hercules) Initialize() {
	log.Debug().Msg("initializing Hercules")
	d.configure()
	d.initializeFlock()
	d.loadPackages()
	d.initializePackages()
	d.initializeRegistry()
	log.Debug().Interface("config", d.config).Msg("running with config")
}

func (d *Hercules) Run() {
	mux := http.NewServeMux()
	prometheus.Unregister(collectors.NewGoCollector()) // Remove golang node defaults
	mux.Handle("/metrics", middleware.MetricsMiddleware(d.conn, d.metricRegistry, promhttp.Handler()))

	srv := &http.Server{
		Addr:    ":" + d.config.Port,
		Handler: mux,
	}
	go func() {
		log.Info().Msg("Hercules is running with version: " + VERSION)
		if err := srv.ListenAndServe(); err != nil && errors.Is(err, http.ErrServerClosed) {
			log.Info().Msgf("server shut down")
		}
	}()
	// Safe shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info().Msg("shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal().Stack().Err(err).Msg("server forced to shutdown")
		d.db.Close()
		os.Remove(d.config.Db)
	}
	d.db.Close()
	os.Remove(d.config.Db)
}
