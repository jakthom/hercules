package engine

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/jakthom/hercules/pkg/db"
	"github.com/marcboeker/go-duckdb/v2"
)

type Engine struct {
	db     *sql.DB
	conn   *sql.Conn
	mu     sync.Mutex
	closed bool
}

type Options struct {
	MemoryLimit   string
	Threads       int
	TempDirectory string
}

func Open(ctx context.Context, path string) (*Engine, error) {
	return OpenWithOptions(ctx, path, Options{})
}

func OpenWithOptions(ctx context.Context, path string, options Options) (*Engine, error) {
	connector, err := duckdb.NewConnector(path, nil)
	if err != nil {
		return nil, fmt.Errorf("create DuckDB connector: %w", err)
	}
	database := sql.OpenDB(connector)
	conn, err := database.Conn(ctx)
	if err != nil {
		_ = database.Close()
		return nil, fmt.Errorf("open DuckDB connection: %w", err)
	}
	if err := conn.PingContext(ctx); err != nil {
		_ = conn.Close()
		_ = database.Close()
		return nil, fmt.Errorf("ping DuckDB: %w", err)
	}
	createdEngine := &Engine{db: database, conn: conn}
	if err := createdEngine.configure(ctx, options); err != nil {
		_ = createdEngine.Close()
		return nil, err
	}
	return createdEngine, nil
}

func (e *Engine) configure(ctx context.Context, options Options) error {
	settings := make([]db.SQL, 0, 3)
	if options.MemoryLimit != "" {
		settings = append(settings, db.SQL("set memory_limit = '"+escapeLiteral(options.MemoryLimit)+"';"))
	}
	if options.Threads > 0 {
		settings = append(settings, db.SQL(fmt.Sprintf("set threads = %d;", options.Threads)))
	}
	if options.TempDirectory != "" {
		settings = append(settings, db.SQL("set temp_directory = '"+escapeLiteral(options.TempDirectory)+"';"))
	}
	for _, setting := range settings {
		if err := e.Exec(ctx, setting); err != nil {
			return fmt.Errorf("configure DuckDB resources: %w", err)
		}
	}
	return nil
}

func (e *Engine) Exec(ctx context.Context, query db.SQL) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.closed {
		return errors.New("DuckDB engine is closed")
	}
	if _, err := e.conn.ExecContext(ctx, string(query)); err != nil {
		return err
	}
	return nil
}

func (e *Engine) Query(ctx context.Context, query db.SQL, maxRows int) (db.Table, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.closed {
		return db.Table{}, errors.New("DuckDB engine is closed")
	}
	return db.QueryTable(ctx, e.conn, query, maxRows)
}

func (e *Engine) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.closed {
		return nil
	}
	e.closed = true
	return errors.Join(e.conn.Close(), e.db.Close())
}

func escapeLiteral(value string) string {
	return strings.ReplaceAll(value, "'", "''")
}
