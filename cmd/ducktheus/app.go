package main

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dbecorp/ducktheus_exporter/pkg/config"
	"github.com/dbecorp/ducktheus_exporter/pkg/flock"
	metrics "github.com/dbecorp/ducktheus_exporter/pkg/metrics"
	"github.com/dbecorp/ducktheus_exporter/pkg/middleware"
	"github.com/marcboeker/go-duckdb"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
)

var VERSION string

type DuckTheus struct {
	config   config.Config
	db       *sql.DB
	conn     *sql.Conn
	registry *prometheus.Registry
	sources  []metrics.MetricSource
	metrics  []metrics.Metric
	gauges   map[string]prometheus.Gauge
}

func (d *DuckTheus) configure() {
	log.Debug().Msg("configuring ducktheus")
	// Load application config
	d.config, _ = config.GetConfig()
	d.sources = d.config.Sources
	d.metrics = d.config.Metrics
}

func (d *DuckTheus) initializeDuckDB() {
	connector, err := duckdb.NewConnector("ducktheus.db", func(execer driver.ExecerContext) error {
		var err error
		bootQueries := []string{}

		for _, query := range bootQueries {
			_, err = execer.ExecContext(context.Background(), query, nil)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		log.Fatal().Err(err).Msg("could not initialize duckdb database")
	}
	db := sql.OpenDB(connector)
	conn, err := db.Conn(context.Background())
	if err != nil {
		log.Fatal().Err(err).Msg("could not initialize duckdb connection")
	}
	defer db.Close()
	d.conn = conn
	d.db = db
}

func (d *DuckTheus) initializeSources() {
	// For every source start a timer that refreshes said source
	for _, source := range d.sources {
		// Ensure source is populated
		flock.RefreshSource(d.conn, source)
		// Start a ticker to continuously update source on the predefined interval
		ticker := time.NewTicker(time.Duration(source.RefreshIntervalSeconds) * time.Second)
		done := make(chan bool)
		go func() {
			for {
				select {
				case <-done:
					return
				case <-ticker.C:
					go func(conn *sql.Conn, source metrics.MetricSource) error {
						return flock.RefreshSource(conn, source)
					}(d.conn, source)
				}
			}
		}()
	}
}

func (d *DuckTheus) initializeRegistry() {
	log.Debug().Msg("intializing registry")
	d.registry = prometheus.NewRegistry()
	for _, metric := range d.metrics {
		collector := metric.AsCollector()
		d.registry.MustRegister(metric.AsCollector())
	}
}

func (d *DuckTheus) Initialize() {
	log.Debug().Msg("initializing ducktheus")
	d.configure()
	d.initializeDuckDB()
	d.initializeRegistry()
	log.Debug().Interface("config", d.config).Msg("running with config")
}

func (d *DuckTheus) Run() {
	mux := http.NewServeMux()
	prometheus.Unregister(collectors.NewGoCollector()) // Remove all the golang node defaults
	mux.Handle("/metrics", middleware.MetricsMiddleware(d.conn, d.metrics, promhttp.Handler()))

	srv := &http.Server{
		Addr:    ":" + d.config.Port,
		Handler: mux,
	}
	go func() {
		log.Info().Msg("ducktheus is running...")
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
	}
}
