package storage

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	internal "github.com/wf-pro-dev/devbox/internal/cmd"
	"github.com/wf-pro-dev/devbox/internal/db"
)

//go:embed schema.sql
var schemaSQL string

// Store holds the sqlc query interface and the raw DB handle.
type Store struct {
	Queries *db.Queries
	DB      *sql.DB
}

// Open opens (or creates) the SQLite DB at path, runs the schema migrations,
// and returns a ready Store. WAL mode and foreign keys are always enabled.
func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}

	dsn := fmt.Sprintf("%s?_journal_mode=WAL&_foreign_keys=on", path)
	sqlDB, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	// SQLite performs best with a single writer connection.
	sqlDB.SetMaxOpenConns(1)

	if _, err := sqlDB.Exec(schemaSQL); err != nil {
		return nil, fmt.Errorf("run schema: %w", err)
	}

	log.Printf("storage: database ready at %s", path)
	return &Store{
		Queries: db.New(sqlDB),
		DB:      sqlDB,
	}, nil
}

// ── Resolution helpers ────────────────────────────────────────────────────────

// ResolveFile looks up a file by:
//  1. Exact UUID match
//  2. UUID prefix (minimum 6 chars to avoid false positives)
//  3. Exact path match
//  4. Exact file_name match — errors if ambiguous (multiple files share that name)
//
// This is the single canonical resolver used by every CLI command and handler
// that accepts an <id|path|name> argument.
func (s *Store) ResolveFile(query string) (*db.File, error) {

	// 4. Exact file_name — reject if ambiguous.
	params := db.ListFilesParams{}
	ctx := context.Background()
	files, err := s.Queries.ListFiles(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("list by name: %w", err)
	}

	matches := []db.File{}

	for _, f := range files {
		if f.ID == query || internal.ShortID(f.ID) == query || f.Path == query || f.FileName == query {
			matches = append(matches, f)
		}
	}

	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("no file matching %q", query)
	case 1:
		return &matches[0], nil
	default:
		var paths []string
		for _, f := range matches {
			paths = append(paths, f.Path)
		}
		return nil, fmt.Errorf(
			"ambiguous name %q — %d files match:\n  %s\nUse the full path or ID instead",
			query, len(matches), strings.Join(paths, "\n  "),
		)
	}
}

// ── Misc helpers ──────────────────────────────────────────────────────────────

// NullText returns a *string for nullable DB columns.
// Empty string → nil (SQL NULL).
func NullText(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// StringOrEmpty dereferences a nullable string, returning "" for nil.
func StringOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
