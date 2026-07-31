package runtime

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jakthom/hercules/pkg/config"
	"github.com/jakthom/hercules/pkg/db"
	"github.com/jakthom/hercules/pkg/engine"
	"github.com/jakthom/hercules/pkg/exporter"
	"github.com/jakthom/hercules/pkg/labels"
	"github.com/jakthom/hercules/pkg/metric"
	herculespackage "github.com/jakthom/hercules/pkg/package"
	"github.com/jakthom/hercules/pkg/source"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
)

const (
	serverReadTimeout     = 5 * time.Second
	serverWriteTimeout    = 30 * time.Second
	serverIdleTimeout     = 2 * time.Minute
	serverReadHeaderLimit = 5 * time.Second
	serverMaxHeaderBytes  = 32 * 1024
	maxBearerTokenBytes   = 8 * 1024
)

type Options struct {
	ConfigPath string
	Version    string
}

type Runtime struct {
	loaded             config.Loaded
	version            string
	engine             *engine.Engine
	exporter           *exporter.Collector
	sourceManager      *source.Manager
	registry           *prometheus.Registry
	server             *http.Server
	workerCtx          context.Context
	cancelWorkers      context.CancelFunc
	workerWG           sync.WaitGroup
	lifecycleMu        sync.Mutex
	started            bool
	closing            bool
	shutdownOnce       sync.Once
	shutdownDone       chan struct{}
	shutdownMu         sync.Mutex
	shutdownErr        error
	ready              atomic.Bool
	tlsCertificateFile string
	tlsKeyFile         string
	bearerTokenHash    [sha256.Size]byte
	bearerTokenEnabled bool
}

type httpSecurity struct {
	tlsCertificateFile string
	tlsKeyFile         string
	bearerTokenHash    [sha256.Size]byte
	bearerTokenEnabled bool
}

func Build(ctx context.Context, options Options) (*Runtime, error) {
	loaded, packages, err := compileConfiguration(ctx, options)
	if err != nil {
		return nil, err
	}
	httpSecurity, err := loadHTTPSecurity(loaded)
	if err != nil {
		return nil, err
	}

	databasePath := loaded.Config.DB
	if databasePath != config.DefaultDB && !filepath.IsAbs(databasePath) {
		databasePath = filepath.Join(loaded.BaseDir, databasePath)
	}
	tempDirectory := loaded.Config.DuckDB.TempDirectory
	if tempDirectory != "" && !filepath.IsAbs(tempDirectory) {
		tempDirectory = filepath.Join(loaded.BaseDir, tempDirectory)
	}
	duckDB, err := engine.OpenWithOptions(ctx, databasePath, engine.Options{
		MemoryLimit:   loaded.Config.DuckDB.MemoryLimit,
		Threads:       loaded.Config.DuckDB.Threads,
		TempDirectory: tempDirectory,
	})
	if err != nil {
		return nil, err
	}
	cleanupEngine := true
	defer func() {
		if cleanupEngine {
			_ = duckDB.Close()
		}
	}()

	createdRuntime, err := initializeRuntime(ctx, options, loaded, packages, duckDB, httpSecurity)
	if err != nil {
		return nil, err
	}
	cleanupEngine = false
	return createdRuntime, nil
}

// Check validates configuration, package contents, dependencies, names, variables, and trust policy without opening DuckDB.
func Check(ctx context.Context, options Options) error {
	_, _, err := compileConfiguration(ctx, options)
	return err
}

func compileConfiguration(
	ctx context.Context,
	options Options,
) (config.Loaded, []herculespackage.Package, error) {
	loaded, err := config.Load(options.ConfigPath)
	if err != nil {
		return config.Loaded{}, nil, err
	}
	globalLabels, err := loaded.Config.InstanceLabels()
	if err != nil {
		return config.Loaded{}, nil, fmt.Errorf("resolve instance labels: %w", err)
	}

	packages, err := loadPackages(ctx, loaded, options.Version, globalLabels)
	if err != nil {
		return config.Loaded{}, nil, err
	}
	packages, err = herculespackage.Order(packages)
	if err != nil {
		return config.Loaded{}, nil, fmt.Errorf("compile package order: %w", err)
	}
	extensions, _, _, _ := flattenPackages(packages)
	if err := db.ValidateExtensions(extensions, db.ExtensionPolicy{
		Allowed:        loaded.Config.Security.AllowedExtensions,
		AllowCommunity: loaded.Config.Security.AllowCommunityExtensions,
	}); err != nil {
		return config.Loaded{}, nil, fmt.Errorf("validate DuckDB extensions: %w", err)
	}
	return loaded, packages, nil
}

func initializeRuntime(
	ctx context.Context,
	options Options,
	loaded config.Loaded,
	packages []herculespackage.Package,
	duckDB *engine.Engine,
	security httpSecurity,
) (*Runtime, error) {
	extensions, macros, sources, definitions := flattenPackages(packages)
	initializationContext, cancelInitialization := context.WithTimeout(ctx, loaded.Config.CollectionTimeout())
	defer cancelInitialization()
	if err := db.EnsureExtensions(initializationContext, duckDB, extensions, db.ExtensionPolicy{
		Allowed:        loaded.Config.Security.AllowedExtensions,
		AllowCommunity: loaded.Config.Security.AllowCommunityExtensions,
	}); err != nil {
		return nil, fmt.Errorf("initialize DuckDB extensions: %w", err)
	}
	if err := db.EnsureMacros(initializationContext, duckDB, macros); err != nil {
		return nil, fmt.Errorf("initialize DuckDB macros: %w", err)
	}
	if err := source.Initialize(initializationContext, duckDB, sources); err != nil {
		return nil, fmt.Errorf("initialize sources: %w", err)
	}

	metricExporter, err := exporter.New(
		duckDB,
		definitions,
		loaded.Config.MaxRowsPerMetric,
		loaded.Config.MaxSeries,
	)
	if err != nil {
		return nil, fmt.Errorf("compile metric definitions: %w", err)
	}
	if err := metricExporter.Refresh(initializationContext); err != nil {
		return nil, fmt.Errorf("create initial metric snapshot: %w", err)
	}

	workerCtx, cancelWorkers := context.WithCancel(ctx)
	sourceManager := source.NewManager(workerCtx, duckDB, sources, loaded.Config.CollectionTimeout())
	registry := prometheus.NewRegistry()
	if err := registry.Register(metricExporter); err != nil {
		cancelWorkers()
		return nil, fmt.Errorf("register Hercules metric exporter: %w", err)
	}
	if err := registry.Register(sourceManager); err != nil {
		cancelWorkers()
		return nil, fmt.Errorf("register source manager metrics: %w", err)
	}
	runtimeCollectors := []prometheus.Collector{
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Name:        "hercules_build_info",
			Help:        "Build information for this Hercules process.",
			ConstLabels: prometheus.Labels{"version": displayVersion(options.Version)},
		}, func() float64 { return 1 }),
	}
	for _, collector := range runtimeCollectors {
		if err := registry.Register(collector); err != nil {
			cancelWorkers()
			return nil, fmt.Errorf("register runtime collector: %w", err)
		}
	}

	runtime := &Runtime{
		loaded:             loaded,
		version:            options.Version,
		engine:             duckDB,
		exporter:           metricExporter,
		sourceManager:      sourceManager,
		registry:           registry,
		workerCtx:          workerCtx,
		cancelWorkers:      cancelWorkers,
		shutdownDone:       make(chan struct{}),
		tlsCertificateFile: security.tlsCertificateFile,
		tlsKeyFile:         security.tlsKeyFile,
		bearerTokenHash:    security.bearerTokenHash,
		bearerTokenEnabled: security.bearerTokenEnabled,
	}
	runtime.server = &http.Server{
		Addr:              loaded.Config.Address(),
		Handler:           runtime.Handler(),
		ReadTimeout:       serverReadTimeout,
		WriteTimeout:      serverWriteTimeout,
		IdleTimeout:       serverIdleTimeout,
		ReadHeaderTimeout: serverReadHeaderLimit,
		MaxHeaderBytes:    serverMaxHeaderBytes,
	}
	if security.tlsCertificateFile != "" {
		runtime.server.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	}
	runtime.ready.Store(true)
	return runtime, nil
}

func (r *Runtime) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(r.registry, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	}))
	mux.HandleFunc("/healthz", func(response http.ResponseWriter, _ *http.Request) {
		response.Header().Set("Content-Type", "text/plain; charset=utf-8")
		response.WriteHeader(http.StatusOK)
		_, _ = response.Write([]byte("ok\n"))
	})
	mux.HandleFunc("/readyz", func(response http.ResponseWriter, _ *http.Request) {
		response.Header().Set("Content-Type", "text/plain; charset=utf-8")
		if !r.ready.Load() || !r.exporter.Ready() {
			response.WriteHeader(http.StatusServiceUnavailable)
			_, _ = response.Write([]byte("not ready\n"))
			return
		}
		response.WriteHeader(http.StatusOK)
		_, _ = response.Write([]byte("ready\n"))
	})
	mux.Handle("/", http.RedirectHandler("/metrics", http.StatusSeeOther))
	if !r.bearerTokenEnabled {
		return mux
	}
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/metrics" && !r.authorized(request) {
			response.Header().Set("WWW-Authenticate", "Bearer")
			http.Error(response, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		mux.ServeHTTP(response, request)
	})
}

func (r *Runtime) authorized(request *http.Request) bool {
	provided := sha256.Sum256([]byte(request.Header.Get("Authorization")))
	return subtle.ConstantTimeCompare(provided[:], r.bearerTokenHash[:]) == 1
}

func (r *Runtime) Run(ctx context.Context) error {
	r.lifecycleMu.Lock()
	if r.closing {
		r.lifecycleMu.Unlock()
		return errors.New("hercules runtime is closed")
	}
	if r.started {
		r.lifecycleMu.Unlock()
		return errors.New("hercules runtime already started")
	}
	listener, err := net.Listen("tcp", r.server.Addr)
	if err != nil {
		r.lifecycleMu.Unlock()
		return fmt.Errorf("listen for Hercules HTTP: %w", err)
	}
	r.started = true
	r.lifecycleMu.Unlock()
	r.sourceManager.Start()
	r.workerWG.Add(1)
	go func() {
		defer r.workerWG.Done()
		r.exporter.Run(r.workerCtx, r.loaded.Config.CollectionInterval(), r.loaded.Config.CollectionTimeout())
	}()

	serverErrors := make(chan error, 1)
	go func() {
		if r.tlsCertificateFile != "" {
			serverErrors <- r.server.ServeTLS(listener, r.tlsCertificateFile, r.tlsKeyFile)
			return
		}
		serverErrors <- r.server.Serve(listener)
	}()
	log.Info().Str("address", r.server.Addr).Str("version", displayVersion(r.version)).Msg("Hercules is ready")

	select {
	case <-ctx.Done():
		return nil
	case err := <-serverErrors:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("serve Hercules HTTP: %w", err)
	}
}

func (r *Runtime) Close(ctx context.Context) error {
	r.ready.Store(false)
	r.lifecycleMu.Lock()
	r.closing = true
	r.lifecycleMu.Unlock()
	r.shutdownOnce.Do(func() {
		go r.shutdown(ctx)
	})
	select {
	case <-r.shutdownDone:
		return r.shutdownResult()
	case <-ctx.Done():
		return fmt.Errorf("wait for Hercules shutdown: %w", ctx.Err())
	}
}

func (r *Runtime) shutdown(ctx context.Context) {
	var errs []error
	r.cancelWorkers()
	r.lifecycleMu.Lock()
	started := r.started
	r.lifecycleMu.Unlock()
	if started {
		if err := r.server.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("shutdown HTTP server: %w", err))
			if closeErr := r.server.Close(); closeErr != nil {
				errs = append(errs, fmt.Errorf("force close HTTP server: %w", closeErr))
			}
		}
	}
	r.sourceManager.Close()
	r.workerWG.Wait()
	if err := r.engine.Close(); err != nil {
		errs = append(errs, fmt.Errorf("close DuckDB: %w", err))
	}
	if err := r.removeEphemeralDatabase(); err != nil {
		errs = append(errs, err)
	}
	r.shutdownMu.Lock()
	r.shutdownErr = errors.Join(errs...)
	r.shutdownMu.Unlock()
	close(r.shutdownDone)
}

func (r *Runtime) shutdownResult() error {
	r.shutdownMu.Lock()
	defer r.shutdownMu.Unlock()
	return r.shutdownErr
}

func (r *Runtime) removeEphemeralDatabase() error {
	if !r.loaded.Config.EphemeralDB || r.loaded.Config.DB == config.DefaultDB {
		return nil
	}
	path := r.loaded.Config.DB
	if !filepath.IsAbs(path) {
		path = filepath.Join(r.loaded.BaseDir, path)
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove ephemeral DuckDB %q: %w", path, err)
	}
	return nil
}

func loadHTTPSecurity(loaded config.Loaded) (httpSecurity, error) {
	security := httpSecurity{}
	if loaded.Config.HTTP.TLSCertificateFile != "" {
		security.tlsCertificateFile = configRelativePath(loaded.BaseDir, loaded.Config.HTTP.TLSCertificateFile)
		security.tlsKeyFile = configRelativePath(loaded.BaseDir, loaded.Config.HTTP.TLSKeyFile)
		//nolint:gosec // The operator explicitly selects configuration-relative certificate and key files.
		if _, err := tls.LoadX509KeyPair(security.tlsCertificateFile, security.tlsKeyFile); err != nil {
			return httpSecurity{}, fmt.Errorf("load HTTP TLS certificate: %w", err)
		}
	}
	if loaded.Config.HTTP.BearerTokenFile == "" {
		return security, nil
	}
	tokenPath := configRelativePath(loaded.BaseDir, loaded.Config.HTTP.BearerTokenFile)
	token, err := readBearerToken(tokenPath)
	if err != nil {
		return httpSecurity{}, err
	}
	security.bearerTokenHash = sha256.Sum256([]byte("Bearer " + token))
	security.bearerTokenEnabled = true
	return security, nil
}

func readBearerToken(path string) (string, error) {
	//nolint:gosec // The operator explicitly selects a configuration-relative secret file.
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open HTTP bearer token file: %w", err)
	}
	payload, readErr := io.ReadAll(io.LimitReader(file, maxBearerTokenBytes+1))
	closeErr := file.Close()
	if err := errors.Join(readErr, closeErr); err != nil {
		return "", fmt.Errorf("read HTTP bearer token file: %w", err)
	}
	if len(payload) > maxBearerTokenBytes {
		return "", fmt.Errorf("HTTP bearer token exceeds maximum of %d bytes", maxBearerTokenBytes)
	}
	token := strings.TrimSpace(string(payload))
	if token == "" {
		return "", errors.New("HTTP bearer token must not be empty")
	}
	if strings.ContainsAny(token, " \t\r\n") {
		return "", errors.New("HTTP bearer token must not contain whitespace")
	}
	return token, nil
}

func configRelativePath(baseDir string, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(baseDir, path)
}

func loadPackages(
	ctx context.Context,
	loaded config.Loaded,
	version string,
	globalLabels labels.Labels,
) ([]herculespackage.Package, error) {
	packages := make([]herculespackage.Package, 0, len(loaded.Config.Packages)+1)
	policy := herculespackage.LoadPolicy{
		AllowRemote:   loaded.Config.Security.AllowRemotePackages,
		RequireSHA256: loaded.Config.Security.RequireRemotePackageSHA256,
		AllowedHosts:  loaded.Config.Security.AllowedPackageHosts,
		MaxBytes:      loaded.Config.Security.MaxRemotePackageBytes,
		Offline:       loaded.Config.Security.Offline,
	}
	if loaded.Config.Security.PackageCacheDirectory != "" {
		policy.CacheDir = loaded.Config.Security.PackageCacheDirectory
		if !filepath.IsAbs(policy.CacheDir) {
			policy.CacheDir = filepath.Join(loaded.BaseDir, policy.CacheDir)
		}
	}
	for index, packageReference := range loaded.Config.Packages {
		configuredPackage, err := packageReference.Load(ctx, loaded.BaseDir, nil, policy, version)
		if err != nil {
			return nil, fmt.Errorf("load packages[%d]: %w", index, err)
		}
		configuredPackage.Metadata = metric.Metadata{
			PackageName: configuredPackage.Name,
			Prefix:      packageReference.MetricPrefix,
			Labels:      globalLabels,
		}
		configuredPackage.Metrics.InjectMetadata(configuredPackage.Metadata)
		if err := configuredPackage.Metrics.Validate(); err != nil {
			return nil, fmt.Errorf("validate packages[%d] metric metadata: %w", index, err)
		}
		packages = append(packages, configuredPackage)
	}
	core := herculespackage.Package{
		APIVersion: herculespackage.CurrentAPIVersion,
		Kind:       herculespackage.PackageKind,
		Name:       "core",
		Version:    "1.0.0",
		Extensions: loaded.Config.Extensions,
		Macros:     loaded.Config.Macros,
		Sources:    resolveCoreSourcePaths(loaded.Config.Sources, loaded.BaseDir),
		Metrics:    loaded.Config.Metrics,
		Metadata: metric.Metadata{
			PackageName: "core",
			Labels:      globalLabels,
		},
	}
	core.Metrics.InjectMetadata(core.Metadata)
	if err := core.Validate(version); err != nil {
		return nil, fmt.Errorf("validate core package: %w", err)
	}
	return append(packages, core), nil
}

func resolveCoreSourcePaths(sources []source.Source, baseDir string) []source.Source {
	resolved := append([]source.Source(nil), sources...)
	for index := range resolved {
		configuredSource := &resolved[index]
		if configuredSource.Type == source.SQLSourceType || filepath.IsAbs(configuredSource.Source) {
			continue
		}
		parsed, err := url.Parse(configuredSource.Source)
		if err == nil && parsed.Scheme != "" {
			continue
		}
		configuredSource.Source = filepath.Join(baseDir, configuredSource.Source)
	}
	return resolved
}

func flattenPackages(packages []herculespackage.Package) (
	db.Extensions,
	[]db.Macro,
	[]source.Source,
	[]metric.TypedDefinition,
) {
	var extensions db.Extensions
	var macros []db.Macro
	var sources []source.Source
	var definitions []metric.TypedDefinition
	for _, configuredPackage := range packages {
		extensions.Core = append(extensions.Core, configuredPackage.Extensions.Core...)
		extensions.Community = append(extensions.Community, configuredPackage.Extensions.Community...)
		macros = append(macros, configuredPackage.Macros...)
		sources = append(sources, configuredPackage.Sources...)
		definitions = append(definitions, configuredPackage.Metrics.All()...)
	}
	return extensions, macros, sources, definitions
}

func displayVersion(version string) string {
	if version == "" {
		return "dev"
	}
	return version
}
