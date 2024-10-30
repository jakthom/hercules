package config

import (
	"os"

	"github.com/jakthom/hercules/pkg/db"
	herculespackage "github.com/jakthom/hercules/pkg/herculesPackage"

	"github.com/jakthom/hercules/pkg/labels"
	"github.com/jakthom/hercules/pkg/metrics"
	"github.com/jakthom/hercules/pkg/source"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

const (
	// Config File/Env Defaults
	HERCULES_CONFIG_PATH         string = "HERCULES_CONFIG_PATH"
	DEFAULT_HERCULES_CONFIG_PATH string = "hercules.yml"
	DEBUG                        string = "DEBUG"
	TRACE                        string = "TRACE"
	YAML_CONFIG_TYPE             string = "yaml"
	// Default Config Values
	DEFAULT_DEBUG bool   = false
	DEFAULT_PORT  string = "9999"
	DEFAULT_DB    string = "h.db"
	// Labels
	HERCULES_NAME_LABEL = "hercules"
)

type Config struct {
	Name         string                          `json:"name"`
	Debug        bool                            `json:"debug"`
	Port         string                          `json:"port"`
	Db           string                          `json:"db"`
	GlobalLabels labels.Labels                   `json:"globalLabels"`
	Packages     []herculespackage.PackageConfig `json:"packages"`
	Extensions   db.Extensions                   `json:"extensions"`
	Macros       []db.Macro                      `json:"macros"`
	Sources      []source.Source                 `json:"sources"`
	Metrics      metrics.MetricDefinitions       `json:"metrics"`
}

func (c *Config) InstanceLabels() labels.Labels {
	globalLabels := labels.Labels{}
	globalLabels[HERCULES_NAME_LABEL] = c.Name
	for k, v := range c.GlobalLabels {
		globalLabels[k] = labels.InjectLabelFromEnv(v)
	}
	return globalLabels
}

func (c *Config) Validate() {
	// Passthrough for now - stubbed for config validation
}

// Get configuration. If the specified file cannot be read fall back to sane defaults.
func GetConfig() (Config, error) {
	// Load app config from file
	confPath := os.Getenv(HERCULES_CONFIG_PATH)
	if confPath == "" {
		confPath = DEFAULT_HERCULES_CONFIG_PATH
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
	}
	// Mandatory defaults
	config.Db = DEFAULT_DB
	if err := viper.Unmarshal(config); err != nil {
		log.Error().Stack().Err(err)
	}

	return *config, nil
}
