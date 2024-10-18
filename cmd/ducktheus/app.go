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

	"github.com/dbecorp/ducktheus_exporter/pkg/config"
	"github.com/dbecorp/ducktheus_exporter/pkg/flock"
	metrics "github.com/dbecorp/ducktheus_exporter/pkg/metrics"
	"github.com/dbecorp/ducktheus_exporter/pkg/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
)

var VERSION string

type DuckTheus struct {
	config  config.Config
	db      *sql.DB
	conn    *sql.Conn
	sources []metrics.Source
	metrics []metrics.Metric
	gauges  map[string]*prometheus.GaugeVec
}

func (d *DuckTheus) configure() {
	log.Debug().Msg("configuring ducktheus")
	// Load application config
	d.config, _ = config.GetConfig()
	d.sources = d.config.Sources
	d.metrics = d.config.Metrics
}

func (d *DuckTheus) initializeDuckDB() {
	d.db, d.conn = flock.InitializeDB(d.config)
}

func (d *DuckTheus) initializeSources() {
	for _, source := range d.sources {
		source.InitializeWithConnection(d.conn)
	}
}

func (d *DuckTheus) initializeRegistry() {
	log.Debug().Msg("intializing registry")

	gauges := make(map[string]*prometheus.GaugeVec)

	for _, metric := range d.metrics {
		gauge := metric.AsGaugeVec()
		log.Trace().Interface("gauge", metric.Name).Msg("registering gauge with registry")
		prometheus.MustRegister(gauge)
		gauges[metric.Name] = gauge
	}
	d.gauges = gauges
}

func (d *DuckTheus) Initialize() {
	log.Debug().Msg("initializing ducktheus")
	d.configure()
	d.initializeDuckDB()
	d.initializeSources()
	d.initializeRegistry()
	log.Debug().Interface("config", d.config).Msg("running with config")
}

func (d *DuckTheus) Run() {
	mux := http.NewServeMux()
	prometheus.Unregister(collectors.NewGoCollector()) // Remove all the golang node defaults
	mux.Handle("/metrics", middleware.MetricsMiddleware(d.conn, d.metrics, d.gauges, promhttp.Handler()))

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
