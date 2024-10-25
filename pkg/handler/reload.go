package handler

import (
	"net/http"

	"github.com/dbecorp/hercules/pkg/config"
	registry "github.com/dbecorp/hercules/pkg/metricRegistry"
	"github.com/rs/zerolog/log"
)

func PackageReloadHandler(config config.Config, metricRegistries *map[string]*registry.MetricRegistry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		packageName := r.PathValue("pkg")
		registries := *metricRegistries
		metricRegistry, ok := registries[packageName]
		// Fail with 404 if package doesn't current exist in registry
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			log.Debug().Msg("reload rejected - package not found in registry")
			w.Write([]byte("package not reloaded"))
			return
		}
		// Otherwise proceed to create and registry a new package with the application registry
		// NOTE!!! The refresh goroutines should all be shut down before registering a new registry.
		updatedPackage, err := metricRegistry.Package.Config.GetPackage()
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			log.Debug().Msg("reloaded rejected - package configuration not found")
			w.Write([]byte("package not reloaded"))
			return
		}
		newRegistry := registry.NewMetricRegistryfromPackage(updatedPackage, config.InstanceLabels())
		registries[packageName] = newRegistry
		metricRegistries = &registries
		log.Debug().Msg(packageName + " reloaded successfully")
		w.Write([]byte("package reloaded"))
	}
}
