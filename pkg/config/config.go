package config

import (
	"fmt"
	"os"

	"github.com/jakthom/hercules/pkg/db"
	herculespackage "github.com/jakthom/hercules/pkg/herculesPackage"
	"github.com/jakthom/hercules/pkg/labels"
	"github.com/jakthom/hercules/pkg/metric"
	"github.com/jakthom/hercules/pkg/source"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

const (
	// HerculesConfigPath is the environment variable name for the configuration file path.
	HerculesConfigPath string = "HERCULES_CONFIG_PATH"
	// DefaultHerculesConfigPath is the default path for the Hercules configuration file.
	DefaultHerculesConfigPath string = "hercules.yml"
	// DebugEnvVar is the environment variable name to enable debug mode.
	DebugEnvVar string = "DEBUG"
	// TraceEnvVar is the environment variable name to enable trace mode.
	TraceEnvVar string = "TRACE"
	// YamlConfigType is the configuration type for YAML files.
	YamlConfigType string = "yaml"

	// DefaultDebug is the default debug mode setting.
	DefaultDebug bool = false
	// DefaultPort is the default port for the Hercules server.
	DefaultPort string = "9999"
	// DefaultDB is the default database file path.
	DefaultDB string = "h.db"

	// HerculesNameLabel is the label name used for Hercules instance identification.
	HerculesNameLabel = "hercules"
)

type Config struct {
	Name         string                          `json:"name"`
	Debug        bool                            `json:"debug"`
	Port         string                          `json:"port"`
	DB           string                          `json:"db"`
	GlobalLabels labels.Labels                   `json:"globalLabels"`
	Packages     []herculespackage.PackageConfig `json:"packages"`
	Extensions   db.Extensions                   `json:"extensions"`
	Macros       []db.Macro                      `json:"macros"`
	Sources      []source.Source                 `json:"sources"`
	Metrics      metric.Definitions              `json:"metrics"`
}

func (c *Config) InstanceLabels() labels.Labels {
	globalLabels := labels.Labels{}
	globalLabels[HerculesNameLabel] = c.Name
	for k, v := range c.GlobalLabels {
		globalLabels[k] = labels.InjectLabelFromEnv(v)
	}
	return globalLabels
}

func (c *Config) Validate() {
	// Passthrough for now - stubbed for config validation
}

// GetConfig retrieves the application configuration from file or returns defaults.
// If the specified file cannot be read, it will fall back to sane defaults.
func GetConfig() (Config, error) {
	// Load app config from file
	confPath := os.Getenv(HerculesConfigPath)
	if confPath == "" {
		confPath = DefaultHerculesConfigPath
	}

	config := &Config{}

	// Check if the file exists before attempting to read it
	if _, err := os.Stat(confPath); os.IsNotExist(err) {
		log.Error().Str("path", confPath).Msg("config file does not exist")
		config.Debug = DefaultDebug
		config.Port = DefaultPort
		config.DB = DefaultDB
		return *config, fmt.Errorf("config file not found at path: %s", confPath)
	}

	log.Info().Str("path", confPath).Msg("loading config from file")

	// Try to get configuration from file
	viper.SetConfigFile(confPath)
	viper.SetConfigType(YamlConfigType)
	err := viper.ReadInConfig()
	if err != nil {
		log.Error().Err(err).Str("path", confPath).Msg("could not read config - using defaults")
		config.Debug = DefaultDebug
		config.Port = DefaultPort
	} else {
		log.Debug().Str("path", confPath).Msg("config file loaded successfully")
	}

	// Mandatory defaults
	config.DB = DefaultDB
	unmarshalErr := viper.Unmarshal(config)
	if unmarshalErr != nil {
		log.Error().Err(unmarshalErr).Msg("error unmarshaling config")
	}

	return *config, nil
}
