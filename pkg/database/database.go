package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
	_ "modernc.org/sqlite"
)

// Dialect represents the database engine type.
type Dialect string

const (
	DialectPostgres Dialect = "postgres"
	DialectSQLite   Dialect = "sqlite"
)

// DetectDialect determines the database dialect and DSN from environment variables.
// If DATABASE_URL is set, PostgreSQL is used. Otherwise, SQLite is used as the default.
func DetectDialect() (Dialect, string) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL != "" {
		return DialectPostgres, dbURL
	}

	// No DATABASE_URL set — use SQLite as default
	sqlitePath := os.Getenv("SQLITE_DB_PATH")
	if sqlitePath == "" {
		sqlitePath = "data/config.db"
	}
	return DialectSQLite, sqlitePath
}

// Open opens a database connection for the given dialect and DSN.
func Open(dialect Dialect, dsn string) (*sql.DB, error) {
	switch dialect {
	case DialectPostgres:
		return sql.Open("postgres", dsn)
	case DialectSQLite:
		dir := filepath.Dir(dsn)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create SQLite data directory %s: %w", dir, err)
		}
		connStr := fmt.Sprintf("file:%s?_pragma=journal_mode%%28WAL%%29&_pragma=busy_timeout%%285000%%29&_pragma=foreign_keys%%281%%29", dsn)
		return sql.Open("sqlite", connStr)
	default:
		return nil, fmt.Errorf("unsupported database dialect: %s", dialect)
	}
}

// RunMigrations applies database migrations based on the dialect.
func RunMigrations(dialect Dialect, dsn string, db *sql.DB) error {
	switch dialect {
	case DialectPostgres:
		return runPostgresMigrations(dsn)
	case DialectSQLite:
		return initSQLiteSchema(db)
	default:
		return fmt.Errorf("unsupported database dialect: %s", dialect)
	}
}

func runPostgresMigrations(dbURL string) error {
	migrationsPath := "file://db/migrations"
	m, err := migrate.New(migrationsPath, dbURL)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}
	log.Println("PostgreSQL migrations applied successfully.")
	return nil
}

func initSQLiteSchema(db *sql.DB) error {
	sqlFile := "db/sqlite_init.sql"
	data, err := os.ReadFile(sqlFile)
	if err != nil {
		return fmt.Errorf("failed to read SQLite init SQL from %s: %w", sqlFile, err)
	}
	if _, err := db.Exec(string(data)); err != nil {
		return fmt.Errorf("failed to initialize SQLite schema: %w", err)
	}
	log.Println("SQLite schema initialized successfully.")
	return nil
}
