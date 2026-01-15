package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresConnection manages PostgreSQL connection pool
type PostgresConnection struct {
	Pool *pgxpool.Pool
}

// NewPostgresConnection creates a new PostgreSQL connection
func NewPostgresConnection(dsn string) (*PostgresConnection, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse postgres config: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	return &PostgresConnection{Pool: pool}, nil
}

// Ping verifies the connection is alive
func (p *PostgresConnection) Ping(ctx context.Context) error {
	return p.Pool.Ping(ctx)
}

// Close closes the connection pool
func (p *PostgresConnection) Close() {
	p.Pool.Close()
}
