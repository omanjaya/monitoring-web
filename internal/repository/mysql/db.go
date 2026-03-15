package mysql

import (
	"context"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"

	"github.com/diskominfos-bali/monitoring-website/internal/config"
	"github.com/diskominfos-bali/monitoring-website/pkg/logger"
)

// Connect creates a new database connection using the main config
func Connect(cfg *config.Config) (*sqlx.DB, error) {
	return NewDB(cfg.Database)
}

// NewDB creates a new database connection
func NewDB(cfg config.DatabaseConfig) (*sqlx.DB, error) {
	db, err := sqlx.Connect("mysql", cfg.DSN())
	if err != nil {
		return nil, err
	}

	// Configure connection pool
	db.SetMaxOpenConns(100)
	db.SetMaxIdleConns(50)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(5 * time.Minute)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, err
	}

	logger.Info().Msg("Database connected successfully")
	return db, nil
}
