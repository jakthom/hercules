// Package herculespackage_test contains tests for the herculespackage package
package herculespackage_test

import (
	"testing"

	"github.com/jakthom/hercules/pkg/db"
	herculespackage "github.com/jakthom/hercules/pkg/herculesPackage"
	"github.com/jakthom/hercules/pkg/metric"
	"github.com/jakthom/hercules/pkg/source"
	herculestypes "github.com/jakthom/hercules/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"
)

// TestPackageLoadFromFile tests loading a package from a YAML file.
func TestPackageLoadFromFile(t *testing.T) {
	// Use the Snowflake package which uses snowflake_query_history.parquet in the assets directory
	packagePath := "../../hercules-packages/snowflake/1.0.yml"

	// Configure and load package
	config := herculespackage.PackageConfig{
		Package:      packagePath,
		Variables:    herculespackage.Variables{"env": "test"},
		MetricPrefix: "prefix_",
	}

	pkg, err := config.GetPackage()
	require.NoError(t, err, "Should load package without error")

	// Validate core package attributes
	assert.Equal(t, herculestypes.PackageName("snowflake"), pkg.Name)
	assert.Equal(t, "1.0", pkg.Version)
	assert.Equal(t, "test", pkg.Variables["env"])
	assert.Equal(t, herculestypes.MetricPrefix("prefix_"), pkg.MetricPrefix)

	// Validate macros
	assert.GreaterOrEqual(t, len(pkg.Macros), 1, "Should have at least one macro")
	assert.Contains(t, pkg.Macros[0].SQL, "one() AS (SELECT 1)")

	// Validate sources
	assert.GreaterOrEqual(t, len(pkg.Sources), 1, "Should have at least one source")
	assert.Equal(t, "snowflake_query_history", pkg.Sources[0].Name)
	assert.Equal(t, source.Type("parquet"), pkg.Sources[0].Type)
	assert.Equal(t, "assets/snowflake_query_history.parquet", pkg.Sources[0].Source)
	assert.True(t, pkg.Sources[0].Materialize)
	assert.Equal(t, 5, pkg.Sources[0].RefreshIntervalSeconds)

	// Validate metrics - test one of each type
	// Gauge metrics
	assert.GreaterOrEqual(t, len(pkg.Metrics.Gauge), 1, "Should have at least one gauge metric")
	assert.Equal(t, "query_status_count", pkg.Metrics.Gauge[0].Name)
	assert.Contains(t, pkg.Metrics.Gauge[0].Help, "Queries executed and their associated status")

	// Histogram metrics
	assert.GreaterOrEqual(t, len(pkg.Metrics.Histogram), 1, "Should have at least one histogram metric")
	assert.Equal(t, "query_duration_seconds", pkg.Metrics.Histogram[0].Name)
	assert.Contains(t, pkg.Metrics.Histogram[0].Help, "Histogram of query duration seconds")
	assert.GreaterOrEqual(t, len(pkg.Metrics.Histogram[0].Buckets), 1, "Should have histogram buckets")

	// Summary metrics
	assert.GreaterOrEqual(t, len(pkg.Metrics.Summary), 1, "Should have at least one summary metric")
	assert.Equal(t, "virtual_warehouse_query_duration_seconds", pkg.Metrics.Summary[0].Name)
	assert.Contains(t, pkg.Metrics.Summary[0].Help, "Summary of query duration seconds")
	assert.GreaterOrEqual(t, len(pkg.Metrics.Summary[0].Objectives), 1, "Should have summary objectives")

	// Counter metrics
	assert.GreaterOrEqual(t, len(pkg.Metrics.Counter), 1, "Should have at least one counter metric")
	assert.Equal(t, "queries_executed_count", pkg.Metrics.Counter[0].Name)
	assert.Contains(t, pkg.Metrics.Counter[0].Help, "The count of queries executed by user and warehouse")
}

// TestPackageSerialization tests package serialization and deserialization.
func TestPackageSerialization(t *testing.T) {
	// Create test package
	originalPkg := createMinimalTestPackage()

	// Serialize to YAML
	serialized, err := yaml.Marshal(originalPkg)
	require.NoError(t, err, "Package serialization should succeed")

	// Deserialize from YAML
	var deserializedPkg herculespackage.Package
	err = yaml.Unmarshal(serialized, &deserializedPkg)
	require.NoError(t, err, "Package deserialization should succeed")

	// Validate core attributes
	assert.Equal(t, originalPkg.Name, deserializedPkg.Name, "Package name should match")
	assert.Equal(t, originalPkg.Version, deserializedPkg.Version, "Package version should match")

	// Validate component counts
	assert.Len(t, deserializedPkg.Extensions.Community, 1, "Should have one extension")
	assert.Len(t, deserializedPkg.Macros, 1, "Should have one macro")
	assert.Len(t, deserializedPkg.Sources, 1, "Should have one source")
	assert.Len(t, deserializedPkg.Metrics.Gauge, 1, "Should have one gauge metric")

	// Validate component details
	assert.Equal(t, originalPkg.Extensions.Community[0].Name, deserializedPkg.Extensions.Community[0].Name)
	assert.Equal(t, originalPkg.Macros[0].Name, deserializedPkg.Macros[0].Name)
	assert.Equal(t, originalPkg.Sources[0].Name, deserializedPkg.Sources[0].Name)
	assert.Equal(t, originalPkg.Metrics.Gauge[0].Name, deserializedPkg.Metrics.Gauge[0].Name)
}

// Helper functions

// createMinimalTestPackage creates a minimal test package for testing.
func createMinimalTestPackage() herculespackage.Package {
	return herculespackage.Package{
		Name:    "test-package",
		Version: "1.0",
		Extensions: db.Extensions{
			Community: []db.CommunityExtension{
				{Name: "test_ext"},
			},
		},
		Macros: []db.Macro{
			{Name: "test_macro", SQL: "test_macro()"},
		},
		Sources: []source.Source{
			{
				Name:                   "test_source",
				Type:                   "sql",
				Source:                 "SELECT 1",
				Materialize:            true,
				RefreshIntervalSeconds: 60,
			},
		},
		Metrics: metric.Definitions{
			Gauge: []*metric.Definition{
				{Name: "test_gauge", Help: "Test gauge", SQL: "SELECT 1"},
			},
		},
		Metadata: metric.Metadata{
			PackageName: "test-package",
			Prefix:      "test_",
		},
	}
}
