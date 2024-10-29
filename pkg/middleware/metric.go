package middleware

import (
	"database/sql"
	"net/http"

	registry "github.com/jakthom/hercules/pkg/metricRegistry"
	"github.com/rs/zerolog/log"
)

func MetricsMiddleware(conn *sql.Conn, registries []*registry.MetricRegistry, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, registry := range registries {
			err := registry.MaterializeWithConnection(conn)
			if err != nil {
				log.Debug().Interface("packageName", registry.PackageName).Msg("could not materialize registry")
			}
		}
		next.ServeHTTP(w, r)
	})
}
