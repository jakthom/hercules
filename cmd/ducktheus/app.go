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
	config            config.Config
	db                *sql.DB
	conn              *sql.Conn
	sources           []metrics.Source
	metricDefinitions metrics.MetricDefinitions
	metricRegistry    *metrics.MetricRegistry
	// TODO -> Scrap these in favor of a single prometheus metric registry
	gauges     map[string]*prometheus.GaugeVec
	histograms map[string]*prometheus.HistogramVec
	summaries  map[string]*prometheus.SummaryVec
}

func (d *DuckTheus) configure() {
	log.Debug().Msg("configuring ducktheus")
	// Load application config
	d.config, _ = config.GetConfig()
	d.sources = d.config.Sources
	d.metricDefinitions = d.config.Metrics
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
	// TODO - there's a better way to do and store this. It is very sloppy right now. Will refactor asap
	// Initialize all Gauge metrics
	log.Trace().Msg("initializing gauge metrics")
	gauges := make(map[string]*prometheus.GaugeVec)
	for _, gauge := range d.metricDefinitions.Gauge {
		g := gauge.AsGaugeVec()
		log.Trace().Interface("gauge", gauge.Name).Msg("registering gauge with registry")
		prometheus.MustRegister(g)
		gauges[gauge.Name] = g
	}
	d.gauges = gauges
	// Initialize all Histogram metrics
	log.Trace().Msg("initializing histogram metrics")
	histograms := make(map[string]*prometheus.HistogramVec)
	for _, histogram := range d.metricDefinitions.Histogram {
		h := histogram.AsHistogramVec()
		log.Trace().Interface("histogram", histogram.Name).Msg("registering histogram with registry")
		prometheus.MustRegister(h)
		histograms[histogram.Name] = h
	}
	d.histograms = histograms
	// Initialize all Summary metrics
	log.Trace().Msg("initializing summary metrics")
	summaries := make(map[string]*prometheus.SummaryVec)
	for _, summary := range d.metricDefinitions.Summary {
		s := summary.AsSummaryVec()
		log.Trace().Interface("summary", summary.Name).Msg("registering summary with registry")
		prometheus.MustRegister(s)
		summaries[summary.Name] = s
	}
	d.summaries = summaries
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
	// Remove all the golang node defaults
	prometheus.Unregister(collectors.NewGoCollector())
	// TODO -> Make metrics middleware signature better. Much better.
	mux.Handle("/metrics", middleware.MetricsMiddleware(d.conn, d.metricDefinitions, d.gauges, d.histograms, d.summaries, promhttp.Handler()))

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
