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

	"github.com/dbecorp/ducktheus/pkg/config"
	"github.com/dbecorp/ducktheus/pkg/flock"
	metrics "github.com/dbecorp/ducktheus/pkg/metrics"
	"github.com/dbecorp/ducktheus/pkg/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var VERSION string

type DuckTheus struct {
	config         config.Config
	db             *sql.DB
	conn           *sql.Conn
	metricRegistry *metrics.MetricRegistry
}

func (d *DuckTheus) configure() {
	log.Debug().Msg("configuring ducktheus")
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	debug := os.Getenv(config.DEBUG)
	if debug != "" && (debug == "true" || debug == "1" || debug == "True") {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}
	d.config, _ = config.GetConfig()
}

func (d *DuckTheus) initializeFlock() {
	d.db, d.conn = flock.InitializeDB(d.config)
}

func (d *DuckTheus) initializeSources() {
	for _, source := range d.config.Sources {
		source.InitializeWithConnection(d.conn)
	}
}

func (d *DuckTheus) initializeRegistry() {
	d.metricRegistry = metrics.NewMetricRegistry(d.config.Metrics, d.config.InstanceLabels())
}

func (d *DuckTheus) Initialize() {
	log.Debug().Msg("initializing ducktheus")
	d.configure()
	d.initializeFlock()
	d.initializeSources()
	d.initializeRegistry()
	log.Debug().Interface("config", d.config).Msg("running with config")
}

func (d *DuckTheus) Run() {
	mux := http.NewServeMux()
	prometheus.Unregister(collectors.NewGoCollector()) // Remove golang node defaults
	mux.Handle("/metrics", middleware.MetricsMiddleware(d.conn, d.metricRegistry, promhttp.Handler()))

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
