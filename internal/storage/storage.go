package storage

import (
	"database/sql"
	_ "embed"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
	"github.com/wf-pro-dev/devbox/internal/db"
)

//go:embed schema.sql
var schemaSQL string

// Store holds the sqlc query interface and the raw DB for advanced queries.
type Store struct {
	Queries *db.Queries
	DB      *sql.DB
}

// Open opens (or creates) the SQLite database at path, runs the schema,
// and returns a Store with both the sqlc Queries and the raw *sql.DB.
func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}

	dsn := fmt.Sprintf("%s?_journal_mode=WAL&_foreign_keys=on", path)
	sqlDB, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

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
