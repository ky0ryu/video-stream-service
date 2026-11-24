package db

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Config struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

func NewPool(ctx context.Context, c Config) (*pgxpool.Pool, error) {
	if c.Host == "" || c.User == "" || c.DBName == "" {
		return nil, errors.New("incomplete Postgres config: host, user and database are required")
	}

	if c.SSLMode == "" {
		return nil, errors.New("incomplete Postgres config: sslmode is required")
	}

	port := c.Port
	if port == "" {
		port = "5432"
	}

	connStr := fmt.Sprintf("host=%s port=%s user=%s dbname=%s sslmode=%s",
		kvEscape(c.Host), kvEscape(port), kvEscape(c.User), kvEscape(c.DBName), kvEscape(c.SSLMode))

	cfg, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse connStr: %w", err)
	}

	cfg.ConnConfig.Password = c.Password

	// DB pool configuration
	cfg.MaxConns = 8
	cfg.MinConns = 5
	cfg.MaxConnLifetime = 1 * time.Hour
	cfg.MaxConnIdleTime = 15 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create pool: %w", err)
	}

	// verify DB connectivity
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("DB ping failed: %w", err)
	}

	return pool, nil
}

// kvEscape quotes a value for a libpq keyword/value connection string.
func kvEscape(v string) string {
	// values without spaces or quotes can be written bare
	if !strings.ContainsAny(v, " \t'\\") {
		return v
	}
	// apply single quotes with backslash escapes
	return "'" + strings.NewReplacer(`\`, `\\`, `'`, `\'`).Replace(v) + "'"
}
