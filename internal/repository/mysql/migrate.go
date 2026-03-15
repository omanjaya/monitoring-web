package mysql

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/diskominfos-bali/monitoring-website/pkg/logger"
)

// migrationPattern matches numbered migration files like 001_initial_schema.sql
var migrationPattern = regexp.MustCompile(`^\d{3}_.*\.sql$`)

// ensureMigrationsTable creates the schema_migrations table if it doesn't exist.
func ensureMigrationsTable(db *sqlx.DB) error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			id         BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
			filename   VARCHAR(255) NOT NULL UNIQUE,
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
	`
	_, err := db.Exec(query)
	return err
}

// getAppliedMigrations returns a set of filenames that have already been applied.
func getAppliedMigrations(db *sqlx.DB) (map[string]bool, error) {
	applied := make(map[string]bool)
	var filenames []string
	if err := db.Select(&filenames, "SELECT filename FROM schema_migrations ORDER BY filename"); err != nil {
		return nil, fmt.Errorf("failed to query schema_migrations: %w", err)
	}
	for _, f := range filenames {
		applied[f] = true
	}
	return applied, nil
}

// getMigrationFiles reads and sorts numbered migration SQL files from the given directory.
func getMigrationFiles(migrationsDir string) ([]string, error) {
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read migrations directory %s: %w", migrationsDir, err)
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if migrationPattern.MatchString(entry.Name()) {
			files = append(files, entry.Name())
		}
	}

	sort.Strings(files)
	return files, nil
}

// splitStatements splits a SQL file content into individual statements.
// It strips comment-only lines and splits on semicolons.
func splitStatements(content string) []string {
	// Remove full-line comments first, preserving non-comment content
	var lines []string
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "--") {
			continue
		}
		lines = append(lines, line)
	}
	cleaned := strings.Join(lines, "\n")

	var statements []string
	for _, stmt := range strings.Split(cleaned, ";") {
		stmt = strings.TrimSpace(stmt)
		if stmt != "" {
			statements = append(statements, stmt)
		}
	}
	return statements
}

// RunMigrations reads SQL migration files from the migrations/ directory,
// checks which ones have been applied, and applies the remaining ones in order.
// Each migration is wrapped in a transaction where possible.
func RunMigrations(db *sqlx.DB) error {
	logger.Info().Msg("Starting database migration check")

	// Ensure the schema_migrations tracking table exists
	if err := ensureMigrationsTable(db); err != nil {
		return fmt.Errorf("failed to create schema_migrations table: %w", err)
	}

	// Get already-applied migrations
	applied, err := getAppliedMigrations(db)
	if err != nil {
		return err
	}

	// Determine migrations directory relative to the working directory
	migrationsDir := "migrations"
	// Also check if running from cmd/server
	if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
		// Try relative to executable or common project paths
		execPath, _ := os.Executable()
		if execPath != "" {
			candidate := filepath.Join(filepath.Dir(execPath), "migrations")
			if _, err := os.Stat(candidate); err == nil {
				migrationsDir = candidate
			}
		}
	}

	files, err := getMigrationFiles(migrationsDir)
	if err != nil {
		return err
	}

	if len(files) == 0 {
		logger.Info().Msg("No migration files found")
		return nil
	}

	// Count pending
	pending := 0
	for _, f := range files {
		if !applied[f] {
			pending++
		}
	}

	if pending == 0 {
		logger.Info().
			Int("total", len(files)).
			Msg("All migrations already applied")
		return nil
	}

	logger.Info().
		Int("total", len(files)).
		Int("applied", len(applied)).
		Int("pending", pending).
		Msg("Migrations to apply")

	// Apply each pending migration
	for _, filename := range files {
		if applied[filename] {
			continue
		}

		filePath := filepath.Join(migrationsDir, filename)
		content, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", filename, err)
		}

		logger.Info().Str("file", filename).Msg("Applying migration")
		start := time.Now()

		// Execute within a transaction
		tx, err := db.Beginx()
		if err != nil {
			return fmt.Errorf("failed to begin transaction for %s: %w", filename, err)
		}

		// Split and execute statements individually because MySQL doesn't support
		// multiple statements in a single Exec by default
		statements := splitStatements(string(content))
		for i, stmt := range statements {
			if _, err := tx.Exec(stmt); err != nil {
				// Tolerate "Duplicate column name" (1060) and "Duplicate key name" (1061)
				// errors for ALTER TABLE statements, since columns/indexes may already exist
				errMsg := err.Error()
				if strings.Contains(strings.ToUpper(stmt), "ALTER TABLE") &&
					(strings.Contains(errMsg, "1060") || strings.Contains(errMsg, "1061")) {
					logger.Warn().
						Str("file", filename).
						Int("statement", i+1).
						Str("error", errMsg).
						Msg("Skipping ALTER TABLE statement (already applied)")
					continue
				}
				tx.Rollback()
				return fmt.Errorf("migration %s failed at statement %d: %w\nStatement: %s", filename, i+1, err, truncate(stmt, 200))
			}
		}

		// Record that this migration was applied
		if _, err := tx.Exec("INSERT INTO schema_migrations (filename) VALUES (?)", filename); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to record migration %s: %w", filename, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration %s: %w", filename, err)
		}

		logger.Info().
			Str("file", filename).
			Dur("duration", time.Since(start)).
			Msg("Migration applied successfully")
	}

	logger.Info().
		Int("applied", pending).
		Msg("All pending migrations applied successfully")
	return nil
}

// truncate shortens a string to maxLen characters for logging purposes.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
