package config

import (
	"os"
	"strconv"

	"github.com/dbecorp/ducktheus_exporter/pkg/db"
	"github.com/dbecorp/ducktheus_exporter/pkg/metrics"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

const (
	// Config File/Env Defaults
	DUCKTHEUS_CONFIG_PATH         string = "DUCKTHEUS_CONFIG_PATH"
	DEFAULT_DUCKTHEUS_CONFIG_PATH string = "ducktheus.yml"
	DEBUG                         string = "DEBUG"
	YAML_CONFIG_TYPE              string = "yaml"
	// Default Config Values
	DEFAULT_DEBUG bool   = false
	DEFAULT_PORT  string = "9999"
	DEFAULT_DB    string = "ducktheus.db"
)

type Config struct {
	Debug   bool                   `json:"debug"`
	Port    string                 `json:"port"`
	Db      string                 `json:"db"`
	Macros  []db.Macro             `json:"macros"`
	Sources []metrics.MetricSource `json:"sources"`
	Metrics []metrics.Metric       `json:"metrics"`
}

func (c *Config) Validate() error {
	// Passthrough for now - stubbed for config validation
	return nil
}

// Get configuration. If the specified file cannot be read fall back to sane defaults.
func GetConfig() (Config, error) {
	// Load app config from file
	confPath := os.Getenv(DUCKTHEUS_CONFIG_PATH)
	if confPath == "" {
		confPath = DEFAULT_DUCKTHEUS_CONFIG_PATH
	}
	log.Info().Msg("loading config from " + confPath)
	config := &Config{}
	// Try to get configuration from file
	viper.SetConfigFile(confPath)
	viper.SetConfigType(YAML_CONFIG_TYPE)
	err := viper.ReadInConfig()
	if err != nil {
		log.Error().Stack().Err(err).Msg("could not read config - using defaults")
		config.Debug = DEFAULT_DEBUG
		config.Port = DEFAULT_PORT
		config.Db = DEFAULT_DB
	}
	if err := viper.Unmarshal(config); err != nil {
		log.Error().Stack().Err(err)
	}

	// Env-based overrides
	debugFromEnv := os.Getenv(DEBUG)
	if debugFromEnv != "" {
		debug, err := strconv.ParseBool(os.Getenv(DEBUG))
		if err != nil {
			log.Error().Msg("could set debug from env")
		} else {
			config.Debug = debug
		}
	}

	return *config, nil
}
