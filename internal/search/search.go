// Package search implements full-text search over files and collections.
// FTS5 virtual table syntax cannot be validated by sqlc so queries are
// written directly against *sql.DB here.
package search

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

// SearchFiles runs an FTS5 MATCH query over files.
func (s *Searcher) Search(ctx context.Context, query string) ([]db.File, error) {
	return s.searchFiles(ctx, query)
}

func (s *Searcher) searchFiles(ctx context.Context, query string) ([]db.File, error) {
	const q = `
		SELECT f.id, f.path, f.file_name, f.description,
		       f.language, f.size, f.sha256, f.uploaded_by,
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
			&f.ID, &f.Path, &f.FileName,
			&f.Description, &f.Language, &f.Size, &f.Sha256,
			&f.UploadedBy, &f.Version, &f.CreatedAt, &f.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan file: %w", err)
		}
		files = append(files, f)
	}
	return files, rows.Err()
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
