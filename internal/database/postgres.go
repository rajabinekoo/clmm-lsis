package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// PostgresOptions controls the database/sql connection pool.
//
// The research CLI is not a long-running web service. Conservative connection
// limits reduce load on the user's existing research database.
type PostgresOptions struct {
	MaxOpenConnections int
	MaxIdleConnections int

	ConnectionMaxLifetime time.Duration
	ConnectionMaxIdleTime time.Duration

	PingTimeout time.Duration
}

// DefaultPostgresOptions returns conservative settings suitable for
// deterministic offline research commands.
func DefaultPostgresOptions() PostgresOptions {
	return PostgresOptions{
		MaxOpenConnections: 4,
		MaxIdleConnections: 4,

		ConnectionMaxLifetime: 30 * time.Minute,
		ConnectionMaxIdleTime: 5 * time.Minute,

		PingTimeout: 10 * time.Second,
	}
}

// OpenPostgres opens and verifies a PostgreSQL database connection.
//
// The DSN is never included in returned errors because it may contain database
// credentials.
func OpenPostgres(
	ctx context.Context,
	dsn string,
	options PostgresOptions,
) (*sql.DB, error) {
	if strings.TrimSpace(dsn) == "" {
		return nil, fmt.Errorf(
			"open PostgreSQL: database URL is empty",
		)
	}

	if err := options.Validate(); err != nil {
		return nil, fmt.Errorf(
			"open PostgreSQL: %w",
			err,
		)
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf(
			"open PostgreSQL driver: %w",
			err,
		)
	}

	db.SetMaxOpenConns(
		options.MaxOpenConnections,
	)

	db.SetMaxIdleConns(
		options.MaxIdleConnections,
	)

	db.SetConnMaxLifetime(
		options.ConnectionMaxLifetime,
	)

	db.SetConnMaxIdleTime(
		options.ConnectionMaxIdleTime,
	)

	pingContext, cancel := context.WithTimeout(
		ctx,
		options.PingTimeout,
	)
	defer cancel()

	if err := db.PingContext(pingContext); err != nil {
		_ = db.Close()

		return nil, fmt.Errorf(
			"ping PostgreSQL: %w",
			err,
		)
	}

	return db, nil
}

func (o PostgresOptions) Validate() error {
	if o.MaxOpenConnections <= 0 {
		return fmt.Errorf(
			"maximum open connections must be greater than zero",
		)
	}

	if o.MaxIdleConnections < 0 {
		return fmt.Errorf(
			"maximum idle connections must not be negative",
		)
	}

	if o.MaxIdleConnections >
		o.MaxOpenConnections {
		return fmt.Errorf(
			"maximum idle connections must not exceed maximum open connections",
		)
	}

	if o.ConnectionMaxLifetime < 0 {
		return fmt.Errorf(
			"connection maximum lifetime must not be negative",
		)
	}

	if o.ConnectionMaxIdleTime < 0 {
		return fmt.Errorf(
			"connection maximum idle time must not be negative",
		)
	}

	if o.PingTimeout <= 0 {
		return fmt.Errorf(
			"ping timeout must be greater than zero",
		)
	}

	return nil
}
