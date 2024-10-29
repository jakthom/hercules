package handler

import (
	"encoding/json"
	"net/http"
	"sort"

	"github.com/dbecorp/hercules/pkg/config"
	registry "github.com/dbecorp/hercules/pkg/metricRegistry"
	"github.com/rs/zerolog/log"
)

const HTTP_PACKAGE_DIRECTORY_ROUTE string = "/packages/"
const HTTP_PACKAGE_RELOAD_ROUTE string = "/packages/{pkg}/reload"

func PackageDirectoryHandler(metricRegistries *map[string]*registry.MetricRegistry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		packageNames := make([]string, 0)
		for k := range *metricRegistries {
			packageNames = append(packageNames, k)
		}
		sort.Strings(packageNames)
		jsonResp, err := json.Marshal(packageNames)
		if err != nil {
			log.Debug().Msg("could not marshal package names")
		}
		w.Write(jsonResp)
	}
}

func PackageReloadHandler(config config.Config, metricRegistries *map[string]*registry.MetricRegistry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		packageName := r.PathValue("pkg")
		registries := *metricRegistries
		metricRegistry, ok := registries[packageName]
		// Fail with 404 if package doesn't current exist in registry
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			log.Debug().Interface("package", packageName).Msg("reload rejected - package not found in registry")
			w.Write([]byte("package not reloaded"))
			return
		}
		// Otherwise proceed to create and registry a new package with the application registry
		// NOTE!!! The source refresh goroutines NEED all be shut down before registering a new registry. Otherwise they will be duplicated and become out of sync.
		updatedPackage, err := metricRegistry.Package.Config.GetPackage()
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			log.Debug().Interface("package", packageName).Msg("reloaded rejected - package configuration not found")
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
