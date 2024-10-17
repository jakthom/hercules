package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dbecorp/ducktheus_exporter/pkg/config"
	"github.com/dbecorp/ducktheus_exporter/pkg/metric"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
)

var VERSION string

type DuckTheus struct {
	config  config.Config
	sources map[string]metric.MetricSource
	metrics map[string]metric.Metric
}

func (d *DuckTheus) configure() {
	log.Debug().Msg("configuring ducktheus")
	// Load application config
	d.config, _ = config.GetConfig()
}

func (d *DuckTheus) Initialize() {
	log.Debug().Msg("initializing ducktheus")
	d.configure()
	// Remove
	log.Debug().Interface("config", d.config).Msg("running with config")
}

func (d *DuckTheus) Run() {
	mux := http.NewServeMux()
	prometheus.Unregister(collectors.NewGoCollector()) // Remove all the golang node defaults
	mux.Handle("/metrics", promhttp.Handler())

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
