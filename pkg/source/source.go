package source

import (
	"context"
	"fmt"
	"hash/fnv"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/jakthom/hercules/pkg/db"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
)

type Type string

const (
	SQLSourceType     Type = "sql"
	ParquetSourceType Type = "parquet"
	JSONSourceType    Type = "json"
	CSVSourceType     Type = "csv"
	maxRefreshSeconds      = 7 * 24 * 60 * 60
)

var sourceNamePattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

type Source struct {
	Name                   string `json:"name"`
	Type                   Type   `json:"type"`
	Source                 string `json:"source"`
	Materialize            bool   `json:"materialize"`
	RefreshIntervalSeconds int    `json:"refreshIntervalSeconds"`
}

type Status struct {
	LastSuccess time.Time
	LastError   string
	Duration    time.Duration
	Refreshing  bool
	Failures    uint64
}

type Manager struct {
	executor db.Executor
	sources  []Source
	timeout  time.Duration

	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	lifecycleMu sync.Mutex
	started     bool
	closed      bool
	statusMu    sync.RWMutex
	statuses    map[string]Status

	refreshingDesc  *prometheus.Desc
	durationDesc    *prometheus.Desc
	lastSuccessDesc *prometheus.Desc
	failuresDesc    *prometheus.Desc
}

func (s Source) Validate() error {
	if !sourceNamePattern.MatchString(s.Name) {
		return fmt.Errorf("invalid source name %q", s.Name)
	}
	if strings.TrimSpace(s.Source) == "" {
		return fmt.Errorf("source %q has empty source", s.Name)
	}
	switch s.Type {
	case SQLSourceType, ParquetSourceType, JSONSourceType, CSVSourceType:
	default:
		return fmt.Errorf("source %q has unsupported type %q", s.Name, s.Type)
	}
	if s.RefreshIntervalSeconds < 0 {
		return fmt.Errorf("source %q refreshIntervalSeconds must not be negative", s.Name)
	}
	if s.RefreshIntervalSeconds > maxRefreshSeconds {
		return fmt.Errorf("source %q refreshIntervalSeconds must not exceed %d", s.Name, maxRefreshSeconds)
	}
	if !s.Materialize && s.RefreshIntervalSeconds > 0 {
		return fmt.Errorf("view source %q must not configure refreshIntervalSeconds", s.Name)
	}
	return nil
}

func (s Source) SQL() db.SQL {
	escaped := strings.ReplaceAll(s.Source, "'", "''")
	switch s.Type {
	case ParquetSourceType:
		return db.SQL(fmt.Sprintf("select * from read_parquet('%s')", escaped))
	case CSVSourceType:
		return db.SQL(fmt.Sprintf("select * from read_csv_auto('%s')", escaped))
	case JSONSourceType:
		return db.SQL(fmt.Sprintf("select * from read_json_auto('%s')", escaped))
	case SQLSourceType:
		return db.SQL(s.Source)
	default:
		return ""
	}
}

func (s Source) CreateOrReplaceSQL() db.SQL {
	kind := "view"
	if s.Materialize {
		kind = "table"
	}
	return db.SQL(fmt.Sprintf("create or replace %s %s as %s;", kind, quoteIdentifier(s.Name), s.SQL()))
}

func Initialize(ctx context.Context, executor db.Executor, sources []Source) error {
	seen := make(map[string]struct{}, len(sources))
	for _, configuredSource := range sources {
		if err := configuredSource.Validate(); err != nil {
			return err
		}
		lookupName := strings.ToLower(configuredSource.Name)
		if _, exists := seen[lookupName]; exists {
			return fmt.Errorf("duplicate source name %q", configuredSource.Name)
		}
		seen[lookupName] = struct{}{}
		if err := executor.Exec(ctx, configuredSource.CreateOrReplaceSQL()); err != nil {
			return fmt.Errorf("initialize source %q: %w", configuredSource.Name, err)
		}
	}
	return nil
}

func NewManager(parent context.Context, executor db.Executor, sources []Source, timeout time.Duration) *Manager {
	ctx, cancel := context.WithCancel(parent)
	manager := &Manager{
		executor: executor,
		sources:  append([]Source(nil), sources...),
		timeout:  timeout,
		ctx:      ctx,
		cancel:   cancel,
		statuses: make(map[string]Status, len(sources)),
		refreshingDesc: prometheus.NewDesc(
			"hercules_source_refreshing",
			"Whether a Hercules source refresh is currently running.",
			[]string{"source"}, nil,
		),
		durationDesc: prometheus.NewDesc(
			"hercules_source_refresh_duration_seconds",
			"Duration of the most recent Hercules source refresh.",
			[]string{"source"}, nil,
		),
		lastSuccessDesc: prometheus.NewDesc(
			"hercules_source_last_success_timestamp_seconds",
			"Unix timestamp of the most recent successful Hercules source refresh.",
			[]string{"source"}, nil,
		),
		failuresDesc: prometheus.NewDesc(
			"hercules_source_refresh_failures_total",
			"Total failed Hercules source refreshes.",
			[]string{"source"}, nil,
		),
	}
	for _, configuredSource := range sources {
		if configuredSource.Materialize && configuredSource.RefreshIntervalSeconds > 0 {
			manager.statuses[configuredSource.Name] = Status{}
		}
	}
	return manager
}

func (m *Manager) Describe(ch chan<- *prometheus.Desc) {
	ch <- m.refreshingDesc
	ch <- m.durationDesc
	ch <- m.lastSuccessDesc
	ch <- m.failuresDesc
}

func (m *Manager) Collect(ch chan<- prometheus.Metric) {
	for name, status := range m.Statuses() {
		refreshing := 0.0
		if status.Refreshing {
			refreshing = 1
		}
		lastSuccess := 0.0
		if !status.LastSuccess.IsZero() {
			lastSuccess = float64(status.LastSuccess.Unix())
		}
		ch <- prometheus.MustNewConstMetric(m.refreshingDesc, prometheus.GaugeValue, refreshing, name)
		ch <- prometheus.MustNewConstMetric(m.durationDesc, prometheus.GaugeValue, status.Duration.Seconds(), name)
		ch <- prometheus.MustNewConstMetric(m.lastSuccessDesc, prometheus.GaugeValue, lastSuccess, name)
		ch <- prometheus.MustNewConstMetric(m.failuresDesc, prometheus.CounterValue, float64(status.Failures), name)
	}
}

func (m *Manager) Start() {
	m.lifecycleMu.Lock()
	defer m.lifecycleMu.Unlock()
	if m.started || m.closed {
		return
	}
	m.started = true
	for _, configuredSource := range m.sources {
		if !configuredSource.Materialize || configuredSource.RefreshIntervalSeconds <= 0 {
			continue
		}
		m.wg.Add(1)
		go m.run(configuredSource)
	}
}

func (m *Manager) Close() {
	m.lifecycleMu.Lock()
	if m.closed {
		m.lifecycleMu.Unlock()
		return
	}
	m.closed = true
	m.cancel()
	m.lifecycleMu.Unlock()
	m.wg.Wait()
}

func (m *Manager) Statuses() map[string]Status {
	m.statusMu.RLock()
	defer m.statusMu.RUnlock()
	result := make(map[string]Status, len(m.statuses))
	for name, status := range m.statuses {
		result[name] = status
	}
	return result
}

func (m *Manager) run(configuredSource Source) {
	defer m.wg.Done()
	interval := time.Duration(configuredSource.RefreshIntervalSeconds) * time.Second
	timer := time.NewTimer(jitter(configuredSource.Name, interval))
	defer timer.Stop()
	backoff := interval
	for {
		select {
		case <-m.ctx.Done():
			return
		case <-timer.C:
			err := m.refresh(configuredSource)
			if err != nil {
				log.Error().Err(err).Str("source", configuredSource.Name).Msg("source refresh failed")
				backoff = min(backoff*2, interval*8)
			} else {
				backoff = interval
			}
			timer.Reset(jitter(configuredSource.Name, backoff))
		}
	}
}

func (m *Manager) refresh(configuredSource Source) error {
	started := time.Now()
	m.setStatus(configuredSource.Name, func(status Status) Status {
		status.Refreshing = true
		return status
	})
	ctx, cancel := context.WithTimeout(m.ctx, m.timeout)
	defer cancel()
	err := m.executor.Exec(ctx, configuredSource.CreateOrReplaceSQL())
	duration := time.Since(started)
	m.setStatus(configuredSource.Name, func(status Status) Status {
		status.Refreshing = false
		status.Duration = duration
		if err != nil {
			status.LastError = err.Error()
			status.Failures++
		} else {
			status.LastError = ""
			status.LastSuccess = time.Now()
		}
		return status
	})
	if err != nil {
		return fmt.Errorf("refresh source %q: %w", configuredSource.Name, err)
	}
	return nil
}

func (m *Manager) setStatus(name string, update func(Status) Status) {
	m.statusMu.Lock()
	defer m.statusMu.Unlock()
	m.statuses[name] = update(m.statuses[name])
}

func jitter(name string, duration time.Duration) time.Duration {
	if duration <= 0 {
		return duration
	}
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(name))
	percent := int(hash.Sum32()%21) - 10
	return duration + time.Duration(percent)*duration/100
}

func quoteIdentifier(identifier string) string {
	return `"` + strings.ReplaceAll(identifier, `"`, `""`) + `"`
}
