package service

import (
	"testing"
	"time"

	"github.com/digitalsolutionsai/scope-config-service/pkg/database"
)

func TestArrayParam_Postgres(t *testing.T) {
	val, err := arrayParam(database.DialectPostgres, []string{"a", "b"})
	if err != nil {
		t.Fatalf("arrayParam(postgres) error: %v", err)
	}
	if val == nil {
		t.Fatal("expected non-nil value for postgres array param")
	}
}

func TestArrayParam_SQLite(t *testing.T) {
	val, err := arrayParam(database.DialectSQLite, []string{"SYSTEM", "PROJECT"})
	if err != nil {
		t.Fatalf("arrayParam(sqlite) error: %v", err)
	}
	s, ok := val.(string)
	if !ok {
		t.Fatalf("expected string for sqlite, got %T", val)
	}
	if s != `["SYSTEM","PROJECT"]` {
		t.Errorf("arrayParam(sqlite) = %q, want %q", s, `["SYSTEM","PROJECT"]`)
	}
}

func TestArrayParam_SQLiteEmpty(t *testing.T) {
	val, err := arrayParam(database.DialectSQLite, []string{})
	if err != nil {
		t.Fatalf("arrayParam(sqlite, empty) error: %v", err)
	}
	s, ok := val.(string)
	if !ok {
		t.Fatalf("expected string, got %T", val)
	}
	if s != "[]" {
		t.Errorf("arrayParam(sqlite, empty) = %q, want %q", s, "[]")
	}
}

func TestArrayScanner_SQLite(t *testing.T) {
	var result []string
	scanner := newArrayScanner(database.DialectSQLite, &result)

	err := scanner.Scan(`["SYSTEM","USER"]`)
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}
	if len(result) != 2 || result[0] != "SYSTEM" || result[1] != "USER" {
		t.Errorf("Scan result = %v, want [SYSTEM USER]", result)
	}
}

func TestArrayScanner_SQLiteNil(t *testing.T) {
	var result []string
	scanner := newArrayScanner(database.DialectSQLite, &result)

	err := scanner.Scan(nil)
	if err != nil {
		t.Fatalf("Scan(nil) error: %v", err)
	}
	if result != nil {
		t.Errorf("Scan(nil) result = %v, want nil", result)
	}
}

func TestArrayScanner_SQLiteBytes(t *testing.T) {
	var result []string
	scanner := newArrayScanner(database.DialectSQLite, &result)

	err := scanner.Scan([]byte(`["A","B"]`))
	if err != nil {
		t.Fatalf("Scan([]byte) error: %v", err)
	}
	if len(result) != 2 || result[0] != "A" || result[1] != "B" {
		t.Errorf("Scan([]byte) result = %v, want [A B]", result)
	}
}

func TestArrayScanner_UnsupportedType(t *testing.T) {
	var result []string
	scanner := newArrayScanner(database.DialectSQLite, &result)

	err := scanner.Scan(12345)
	if err == nil {
		t.Error("expected error for unsupported type, got nil")
	}
}

func TestFlexTime_ScanTime(t *testing.T) {
	ft := &flexTime{}
	now := time.Now()
	err := ft.Scan(now)
	if err != nil {
		t.Fatalf("Scan(time.Time) error: %v", err)
	}
	if !ft.Time.Equal(now) {
		t.Errorf("Time = %v, want %v", ft.Time, now)
	}
}

func TestFlexTime_ScanString(t *testing.T) {
	ft := &flexTime{}
	err := ft.Scan("2026-01-15 10:30:00")
	if err != nil {
		t.Fatalf("Scan(string) error: %v", err)
	}
	expected := time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)
	if !ft.Time.Equal(expected) {
		t.Errorf("Time = %v, want %v", ft.Time, expected)
	}
}

func TestFlexTime_ScanRFC3339(t *testing.T) {
	ft := &flexTime{}
	err := ft.Scan("2026-03-23T14:30:00Z")
	if err != nil {
		t.Fatalf("Scan(RFC3339) error: %v", err)
	}
	expected := time.Date(2026, 3, 23, 14, 30, 0, 0, time.UTC)
	if !ft.Time.Equal(expected) {
		t.Errorf("Time = %v, want %v", ft.Time, expected)
	}
}

func TestFlexTime_ScanNil(t *testing.T) {
	ft := &flexTime{}
	err := ft.Scan(nil)
	if err == nil {
		t.Error("expected error for nil scan on non-nullable flexTime")
	}
}

func TestFlexTime_ScanUnsupportedType(t *testing.T) {
	ft := &flexTime{}
	err := ft.Scan(12345)
	if err == nil {
		t.Error("expected error for unsupported type")
	}
}

func TestFlexTime_ScanInvalidString(t *testing.T) {
	ft := &flexTime{}
	err := ft.Scan("not-a-date")
	if err == nil {
		t.Error("expected error for invalid date string")
	}
}

func TestFlexNullTime_ScanTime(t *testing.T) {
	fnt := &flexNullTime{}
	now := time.Now()
	err := fnt.Scan(now)
	if err != nil {
		t.Fatalf("Scan(time.Time) error: %v", err)
	}
	if !fnt.Valid || !fnt.Time.Equal(now) {
		t.Errorf("Valid=%v, Time=%v, want Valid=true, Time=%v", fnt.Valid, fnt.Time, now)
	}
}

func TestFlexNullTime_ScanNil(t *testing.T) {
	fnt := &flexNullTime{}
	err := fnt.Scan(nil)
	if err != nil {
		t.Fatalf("Scan(nil) error: %v", err)
	}
	if fnt.Valid {
		t.Error("expected Valid=false for nil scan")
	}
}

func TestFlexNullTime_ScanEmptyString(t *testing.T) {
	fnt := &flexNullTime{}
	err := fnt.Scan("")
	if err != nil {
		t.Fatalf("Scan('') error: %v", err)
	}
	if fnt.Valid {
		t.Error("expected Valid=false for empty string")
	}
}

func TestFlexNullTime_ScanString(t *testing.T) {
	fnt := &flexNullTime{}
	err := fnt.Scan("2026-01-15 10:30:00")
	if err != nil {
		t.Fatalf("Scan(string) error: %v", err)
	}
	if !fnt.Valid {
		t.Error("expected Valid=true")
	}
	expected := time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)
	if !fnt.Time.Equal(expected) {
		t.Errorf("Time = %v, want %v", fnt.Time, expected)
	}
}

func TestFlexNullTime_ScanUnsupportedType(t *testing.T) {
	fnt := &flexNullTime{}
	err := fnt.Scan(12345)
	if err == nil {
		t.Error("expected error for unsupported type")
	}
}

func TestParseTimeString_AllFormats(t *testing.T) {
	tests := []struct {
		input string
		year  int
		month time.Month
		day   int
	}{
		{"2026-01-15 10:30:00", 2026, time.January, 15},
		{"2026-03-23T14:30:00Z", 2026, time.March, 23},
		{"2026-06-01T12:00:00+00:00", 2026, time.June, 1},
		{"2026-12-25 08:00:00+00:00", 2026, time.December, 25},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := parseTimeString(tt.input)
			if err != nil {
				t.Fatalf("parseTimeString(%q) error: %v", tt.input, err)
			}
			if result.Year() != tt.year || result.Month() != tt.month || result.Day() != tt.day {
				t.Errorf("parseTimeString(%q) = %v, want year=%d month=%v day=%d", tt.input, result, tt.year, tt.month, tt.day)
			}
		})
	}
}

func TestParseTimeString_Invalid(t *testing.T) {
	_, err := parseTimeString("garbage")
	if err == nil {
		t.Error("expected error for invalid time string")
	}
}
