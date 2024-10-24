package herculespackage

import (
	"database/sql"

	"github.com/dbecorp/hercules/pkg/db"
	"github.com/dbecorp/hercules/pkg/metrics"
	"github.com/dbecorp/hercules/pkg/source"
	herculestypes "github.com/dbecorp/hercules/pkg/types"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

type HerculesPackageVariables map[string]interface{}

// A hercules package consists of extensions, macros, sources, and metrics
// It can be downloaded from remote sources or shipped alongside hercules.

type Package struct {
	Name         herculestypes.PackageName  `json:"name"`
	Version      string                     `json:"version"`
	Variables    HerculesPackageVariables   `json:"variables"`
	MetricPrefix herculestypes.MetricPrefix `json:"metricPrefix"`
	Extensions   db.Extensions              `json:"extensions"`
	Macros       []db.Macro                 `json:"macros"`
	Sources      []source.Source            `json:"sources"`
	Metrics      metrics.MetricDefinitions  `json:"metrics"`
	// TODO -> Package-level secrets
}

func (p *Package) InitializeWithConnection(conn *sql.Conn) error {
	if len(p.Name) > 0 {
		log.Info().Interface("package", p.Name).Msg("initializing " + string(p.Name) + " package")
		// Ensure extensions
		db.EnsureExtensionsWithConnection(p.Extensions, conn)
		// Ensure macros
		db.EnsureMacrosWithConnection(p.Macros, conn)
		// Ensure sources
		source.InitializeSourcesWithConnection(p.Sources, conn)
		log.Info().Interface("package", p.Name).Msg(string(p.Name) + " package initialized")
	} else {
		log.Trace().Msg("empty package detected - skipping initialization")
	}
	return nil
}

type PackageConfig struct {
	Package      string                     `json:"package"`
	Variables    HerculesPackageVariables   `json:"variables"`
	MetricPrefix herculestypes.MetricPrefix `json:"metricPrefix"`
}

func (p *PackageConfig) GetPackage() (Package, error) {
	pkg := &Package{}
	pkg.Variables = p.Variables
	pkg.MetricPrefix = p.MetricPrefix
	// Try to get configuration from file
	viper.SetConfigFile(p.Package)
	viper.SetConfigType("yaml")
	err := viper.ReadInConfig()
	if err != nil {
		log.Debug().Stack().Err(err).Msg("could not load package from location " + p.Package)
		return Package{}, err
	}
	if err := viper.Unmarshal(pkg); err != nil {
		log.Error().Stack().Err(err)
	}
	return *pkg, nil
}
