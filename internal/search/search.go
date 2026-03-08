package search

import (
	"context"
	"database/sql"
	"fmt"
	"io"

	"github.com/wf-pro-dev/devbox/internal/db"
	"github.com/wf-pro-dev/devbox/internal/storage"
)

// Searcher runs FTS5 full-text search queries directly against the raw
// *sql.DB since sqlc cannot validate virtual table syntax.
type Searcher struct {
	sqlDB *sql.DB
}

// New creates a Searcher from the raw sql.DB.
func New(sqlDB *sql.DB) *Searcher {
	return &Searcher{sqlDB: sqlDB}
}

// SearchFiles runs a FTS5 MATCH query and returns matching files by relevance.
func (s *Searcher) SearchFiles(ctx context.Context, query string) ([]db.File, error) {
	const searchSQL = `
		SELECT f.id, f.name, f.description, f.language, f.size,
		       f.blob_path, f.sha256, f.uploaded_by, f.created_at, f.updated_at
		FROM files f
		JOIN files_fts fts ON fts.file_id = f.id
		WHERE files_fts MATCH ?
		ORDER BY rank`

	rows, err := s.sqlDB.QueryContext(ctx, searchSQL, query)
	if err != nil {
		return nil, fmt.Errorf("fts search: %w", err)
	}
	defer rows.Close()

	var files []db.File
	for rows.Next() {
		var f db.File
		if err := rows.Scan(
			&f.ID, &f.Name, &f.Description, &f.Language, &f.Size,
			&f.BlobPath, &f.Sha256, &f.UploadedBy, &f.CreatedAt, &f.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan file: %w", err)
		}
		files = append(files, f)
	}
	return files, rows.Err()
}

// UpdateContent reads the blob for fileID and updates its content in the
// FTS index so the file body becomes searchable.
func (s *Searcher) UpdateContent(ctx context.Context, fileID string, blobs *storage.BlobStore) error {
	f, err := blobs.Read(fileID)
	if err != nil {
		return fmt.Errorf("read blob: %w", err)
	}
	defer f.Close()

	content, err := io.ReadAll(f)
	if err != nil {
		return fmt.Errorf("read content: %w", err)
	}

	_, err = s.sqlDB.ExecContext(ctx,
		`UPDATE files_fts SET content = ? WHERE file_id = ?`,
		string(content), fileID,
	)
	return err
}
