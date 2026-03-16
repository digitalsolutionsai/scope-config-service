package service

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/digitalsolutionsai/scope-config-service/pkg/database"
	"github.com/lib/pq"
)

// arrayParam returns a driver-compatible value for inserting a string array.
// PostgreSQL uses pq.Array; SQLite stores a JSON-encoded text.
func arrayParam(dialect database.Dialect, values []string) interface{} {
	if dialect == database.DialectPostgres {
		return pq.Array(values)
	}
	data, _ := json.Marshal(values)
	return string(data)
}

// arrayScanner implements sql.Scanner for reading a string array column.
type arrayScanner struct {
	dialect database.Dialect
	target  *[]string
}

func newArrayScanner(dialect database.Dialect, target *[]string) *arrayScanner {
	return &arrayScanner{dialect: dialect, target: target}
}

func (a *arrayScanner) Scan(src interface{}) error {
	if a.dialect == database.DialectPostgres {
		return pq.Array(a.target).Scan(src)
	}
	// SQLite: JSON-encoded text
	if src == nil {
		*a.target = nil
		return nil
	}
	var s string
	switch v := src.(type) {
	case string:
		s = v
	case []byte:
		s = string(v)
	default:
		return fmt.Errorf("unsupported type %T for array scan", src)
	}
	return json.Unmarshal([]byte(s), a.target)
}

// flexTime implements sql.Scanner for scanning non-nullable time values from
// both PostgreSQL (time.Time) and SQLite (string) drivers.
type flexTime struct {
	Time time.Time
}

func (ft *flexTime) Scan(value interface{}) error {
	if value == nil {
		return fmt.Errorf("unexpected nil for non-nullable time")
	}
	switch v := value.(type) {
	case time.Time:
		ft.Time = v
		return nil
	case string:
		t, err := parseTimeString(v)
		if err != nil {
			return err
		}
		ft.Time = t
		return nil
	default:
		return fmt.Errorf("unsupported type %T for time scan", value)
	}
}

// flexNullTime implements sql.Scanner for scanning nullable time values from
// both PostgreSQL (time.Time) and SQLite (string) drivers.
type flexNullTime struct {
	Time  time.Time
	Valid bool
}

func (fnt *flexNullTime) Scan(value interface{}) error {
	if value == nil {
		fnt.Valid = false
		return nil
	}
	switch v := value.(type) {
	case time.Time:
		fnt.Time = v
		fnt.Valid = true
		return nil
	case string:
		if v == "" {
			fnt.Valid = false
			return nil
		}
		t, err := parseTimeString(v)
		if err != nil {
			return err
		}
		fnt.Time = t
		fnt.Valid = true
		return nil
	default:
		return fmt.Errorf("unsupported type %T for nullable time scan", value)
	}
}

func parseTimeString(s string) (time.Time, error) {
	layouts := []string{
		"2006-01-02 15:04:05",
		time.RFC3339,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02 15:04:05+00:00",
		"2006-01-02T15:04:05Z",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("unable to parse time string: %s", s)
}
