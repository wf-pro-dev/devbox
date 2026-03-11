// Package search implements full-text search over files and collections.
// FTS5 virtual table syntax cannot be validated by sqlc so queries are
// written directly against *sql.DB here.
package search

import (
	"context"
	"database/sql"

	"github.com/wf-pro-dev/devbox/internal/db"
)

type Searcher struct {
	sqlDB *sql.DB
}

func New(sqlDB *sql.DB) *Searcher {
	return &Searcher{sqlDB: sqlDB}
}

type Results struct {
	Files []db.File `json:"files"`
}

func (s *Searcher) Search(ctx context.Context, query string) (*Results, error) {
	return nil, nil
}

func (s *Searcher) SearchFiles(ctx context.Context, query string) ([]db.File, error) {
	return nil, nil
}

func (s *Searcher) IndexFileContent(ctx context.Context, fileID, content string) error {
	return nil
}

/*
import (
	"context"
	"database/sql"
	"fmt"

	"github.com/wf-pro-dev/devbox/internal/db"
)

// Searcher runs search queries against the raw DB.
type Searcher struct {
	sqlDB *sql.DB
}

// New creates a Searcher.
func New(sqlDB *sql.DB) *Searcher {
	return &Searcher{sqlDB: sqlDB}
}

// Results holds combined search results.
type Results struct {
	Files       []db.File       `json:"files"`
	Collections []db.Collection `json:"collections"`
}

// Search runs a combined search over files (FTS5) and collections (LIKE).
// query is the user-supplied search term.
func (s *Searcher) Search(ctx context.Context, query string) (*Results, error) {
	files, err := s.searchFiles(ctx, query)
	if err != nil {
		return nil, err
	}
	cols, err := s.searchCollections(ctx, query)
	if err != nil {
		return nil, err
	}
	return &Results{Files: files, Collections: cols}, nil
}

// SearchFiles runs an FTS5 MATCH query over files.
func (s *Searcher) SearchFiles(ctx context.Context, query string) ([]db.File, error) {
	return s.searchFiles(ctx, query)
}

func (s *Searcher) searchFiles(ctx context.Context, query string) ([]db.File, error) {
	const q = `
		SELECT f.id, f.path, f.file_name, f.collection_id, f.description,
		       f.language, f.size, f.blob_sha256, f.uploaded_by,
		       f.version, f.created_at, f.updated_at
		FROM files f
		JOIN files_fts fts ON fts.file_id = f.id
		WHERE files_fts MATCH ?
		ORDER BY rank`

	rows, err := s.sqlDB.QueryContext(ctx, q, query)
	if err != nil {
		return nil, fmt.Errorf("fts search files: %w", err)
	}
	defer rows.Close()

	var files []db.File
	for rows.Next() {
		var f db.File
		if err := rows.Scan(
			&f.ID, &f.Path, &f.FileName, &f.CollectionID,
			&f.Description, &f.Language, &f.Size, &f.BlobSha256,
			&f.UploadedBy, &f.Version, &f.CreatedAt, &f.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan file: %w", err)
		}
		files = append(files, f)
	}
	return files, rows.Err()
}

func (s *Searcher) searchCollections(ctx context.Context, query string) ([]db.Collection, error) {
	const q = `
		SELECT id, name, prefix, description, uploaded_by, created_at, updated_at
		FROM collections
		WHERE name LIKE ? OR description LIKE ?
		ORDER BY name`

	like := "%" + query + "%"
	rows, err := s.sqlDB.QueryContext(ctx, q, like, like)
	if err != nil {
		return nil, fmt.Errorf("search collections: %w", err)
	}
	defer rows.Close()

	var cols []db.Collection
	for rows.Next() {
		var c db.Collection
		if err := rows.Scan(
			&c.ID, &c.Name, &c.Prefix, &c.Description,
			&c.UploadedBy, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan collection: %w", err)
		}
		cols = append(cols, c)
	}
	return cols, rows.Err()
}

// IndexFileContent updates the FTS content field for a file.
// Call this after upload or content update so the file body is searchable.
func (s *Searcher) IndexFileContent(ctx context.Context, fileID, content string) error {
	_, err := s.sqlDB.ExecContext(ctx,
		`UPDATE files_fts SET content = ? WHERE file_id = ?`,
		content, fileID,
	)
	return err
}
*/
