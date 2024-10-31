package herculespackage

import (
	"database/sql"
	"io"
	"net/http"
	"os"

	"github.com/jakthom/hercules/pkg/db"
	"github.com/jakthom/hercules/pkg/metrics"
	"github.com/jakthom/hercules/pkg/source"
	herculestypes "github.com/jakthom/hercules/pkg/types"
	"github.com/rs/zerolog/log"
	"sigs.k8s.io/yaml"
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
		err := source.InitializeSourcesWithConnection(p.Sources, conn)
		if err != nil {
			log.Fatal().Interface("package", p.Name).Msg("could not initialize package sources")
		}
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

func (p *PackageConfig) getFromFile() (Package, error) {
	log.Debug().Interface("package", p.Package).Msg("loading " + p.Package + " package from file")
	pkg := Package{}
	yamlFile, err := os.ReadFile(p.Package)
	if err != nil {
		log.Error().Err(err).Msg("could not get package from file " + p.Package)
	}
	err = yaml.Unmarshal(yamlFile, &pkg)
	return pkg, err
}

func (p *PackageConfig) getFromEndpoint() (Package, error) {
	log.Debug().Interface("package", p.Package).Msg("loading " + p.Package + " package from endpoint")
	pkg := Package{}
	resp, err := http.Get(p.Package)
	if err != nil {
		log.Error().Err(err).Msg("could not get package from endpoint " + p.Package)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error().Err(err).Msg("could not read response body from endpoint " + p.Package)
	}
	err = yaml.Unmarshal(body, &pkg)
	return pkg, err
}

func (p *PackageConfig) getFromObjectStorage() (Package, error) {
	log.Fatal().Interface("package", p.Package).Msg("object storage-backed packages are not yet supported")
	pkg := Package{}
	return pkg, nil
}

func (p *PackageConfig) GetPackage() (Package, error) {
	var pkg Package
	var err error
	switch p.Package[0:4] {
	case "http":
		pkg, err = p.getFromEndpoint()
	case "s3:/":
		pkg, err = p.getFromObjectStorage()
	case "gcs:":
		pkg, err = p.getFromObjectStorage()
	default:
		pkg, err = p.getFromFile()
	}
	if err != nil {
		log.Debug().Stack().Err(err).Msg("could not load package from location " + p.Package)
		return Package{}, err
	}
	pkg.Variables = p.Variables
	pkg.MetricPrefix = p.MetricPrefix
	return pkg, nil
}
