package herculespackage

import (
	"github.com/dbecorp/hercules/pkg/db"
	"github.com/dbecorp/hercules/pkg/metrics"
	"github.com/dbecorp/hercules/pkg/source"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

type HerculesPackageVariables map[string]interface{}

// A hercules package consists of extensions, macros, sources, and metrics
// It can be downloaded from remote sources or shipped alongside hercules.

type Package struct {
	Version    string                    `json:"version"`
	Variables  HerculesPackageVariables  `json:"variables"`
	Extensions []db.Extensions           `json:"extensions"`
	Macros     []db.Macro                `json:"macros"`
	Sources    []source.Source           `json:"sources"`
	Metrics    metrics.MetricDefinitions `json:"metrics"`
	// TODO -> Package-level secrets
}

type PackageConfig struct {
	Name      string                   `json:"name"`
	Type      string                   `json:"type"` // s3, gcs, filesystem, etcd, consul
	Location  string                   `json:"location"`
	Variables HerculesPackageVariables `json:"variables"`
}

func (p *PackageConfig) GetPackage() (Package, error) {
	pkg := &Package{}
	// Try to get configuration from file
	viper.SetConfigFile(p.Location)
	viper.SetConfigType("yaml")
	err := viper.ReadInConfig()
	if err != nil {
		log.Error().Stack().Err(err).Msg("could not read config - using defaults")
	}
	if err := viper.Unmarshal(pkg); err != nil {
		log.Error().Stack().Err(err)
	}
	return *pkg, nil
}

// func newPackageFromConfig(config HerculesPackageConfig) HerculesPackage {

// }

// func NewHerculesPackage(config HerculesPackageConfig) HerculesPackage {
// 	pkg
// 	return p
// }
