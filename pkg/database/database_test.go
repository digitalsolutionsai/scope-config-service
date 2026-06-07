package database

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectDialect_Postgres(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgresql://user:pass@localhost:5432/testdb")
	t.Setenv("SQLITE_DB_PATH", "") // should be ignored

	dialect, dsn := DetectDialect()
	if dialect != DialectPostgres {
		t.Errorf("dialect = %q, want %q", dialect, DialectPostgres)
	}
	if dsn != "postgresql://user:pass@localhost:5432/testdb" {
		t.Errorf("dsn = %q, want the DATABASE_URL value", dsn)
	}
}

func TestDetectDialect_SQLiteDefault(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("SQLITE_DB_PATH", "")

	dialect, dsn := DetectDialect()
	if dialect != DialectSQLite {
		t.Errorf("dialect = %q, want %q", dialect, DialectSQLite)
	}
	if dsn != "data/config.db" {
		t.Errorf("dsn = %q, want %q", dsn, "data/config.db")
	}
}

func TestDetectDialect_SQLiteCustomPath(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("SQLITE_DB_PATH", "/tmp/custom.db")

	dialect, dsn := DetectDialect()
	if dialect != DialectSQLite {
		t.Errorf("dialect = %q, want %q", dialect, DialectSQLite)
	}
	if dsn != "/tmp/custom.db" {
		t.Errorf("dsn = %q, want %q", dsn, "/tmp/custom.db")
	}
}

func TestOpen_SQLite(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := Open(DialectSQLite, dbPath)
	if err != nil {
		t.Fatalf("Open(SQLite) failed: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Fatalf("Ping failed: %v", err)
	}
}

func TestOpen_SQLiteCreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	nestedPath := filepath.Join(tmpDir, "sub", "dir", "test.db")

	db, err := Open(DialectSQLite, nestedPath)
	if err != nil {
		t.Fatalf("Open(SQLite) failed: %v", err)
	}
	defer db.Close()

	// Verify directory was created
	dir := filepath.Dir(nestedPath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Errorf("expected directory %q to be created", dir)
	}
}

func TestOpen_UnsupportedDialect(t *testing.T) {
	_, err := Open(Dialect("mysql"), "localhost:3306")
	if err == nil {
		t.Error("expected error for unsupported dialect, got nil")
	}
}

func TestDialectConstants(t *testing.T) {
	if DialectPostgres != "postgres" {
		t.Errorf("DialectPostgres = %q, want %q", DialectPostgres, "postgres")
	}
	if DialectSQLite != "sqlite" {
		t.Errorf("DialectSQLite = %q, want %q", DialectSQLite, "sqlite")
	}
}
