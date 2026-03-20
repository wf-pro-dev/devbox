package storage

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// BlobStore manages raw file content on disk using content-addressable storage.
//
// Layout: <root>/<sha[0:2]>/<sha[2:4]>/<sha>
// Blobs are stored zstd-compressed. Two files with identical content share
// one blob. Ref-counts are maintained in the blobs DB table via triggers —
// the BlobStore only manages the disk side.
type BlobStore struct {
	root string
	db   *sql.DB // used only for ref-count queries
}

// NewBlobStore creates a BlobStore rooted at dir.
func NewBlobStore(root string, db *sql.DB) (*BlobStore, error) {
	if err := os.MkdirAll(root, 0755); err != nil {
		return nil, fmt.Errorf("create blob root: %w", err)
	}
	return &BlobStore{root: root, db: db}, nil
}

// WriteResult is returned by Write.
type WriteResult struct {
	SHA256 string // hex digest of the UNCOMPRESSED content
	Size   int64  // size of the UNCOMPRESSED content in bytes
	Dedupd bool   // true if blob already existed (no disk write performed)
}

// Write reads all of r, computes sha256, and stores the blob compressed on disk
// if it does not already exist (content-addressable deduplication).
// It also inserts a row into the blobs table if this is a new sha256.
// The initial ref_count is set to 0 — the DB trigger on files/versions INSERT
// will increment it when the file row is created.
func (bs *BlobStore) Write(ctx context.Context, r io.Reader) (WriteResult, error) {
	// Buffer the content so we can compute sha256 before writing to disk.
	tmp, err := os.CreateTemp(bs.root, "upload-*")
	if err != nil {
		return WriteResult{}, fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath) // always clean up temp

	h := sha256.New()
	size, err := io.Copy(io.MultiWriter(tmp, h), r)
	if err != nil {
		tmp.Close()
		return WriteResult{}, fmt.Errorf("read content: %w", err)
	}
	tmp.Close()

	digest := hex.EncodeToString(h.Sum(nil))
	result := WriteResult{SHA256: digest, Size: size}

	// If this sha256 already exists in the DB, no disk write needed.
	var existing int
	err = bs.db.QueryRowContext(ctx,
		`SELECT 1 FROM blobs WHERE sha256 = ? LIMIT 1`, digest,
	).Scan(&existing)
	if err == nil {
		// Blob already on disk — just return.
		result.Dedupd = true
		return result, nil
	}
	if err != sql.ErrNoRows {
		return WriteResult{}, fmt.Errorf("check blob exists: %w", err)
	}

	// New blob — compress and write to final CAS path.
	finalPath := bs.Path(digest)
	if err := os.MkdirAll(filepath.Dir(finalPath), 0755); err != nil {
		return WriteResult{}, fmt.Errorf("create blob dir: %w", err)
	}

	// Re-open temp file for reading.
	src, err := os.Open(tmpPath)
	if err != nil {
		return WriteResult{}, fmt.Errorf("reopen temp: %w", err)
	}
	defer src.Close()

	dst, err := os.Create(finalPath)
	if err != nil {
		return WriteResult{}, fmt.Errorf("create blob file: %w", err)
	}
	if _, err := CompressTo(dst, src); err != nil {
		dst.Close()
		os.Remove(finalPath)
		return WriteResult{}, fmt.Errorf("compress blob: %w", err)
	}
	if err := dst.Close(); err != nil {
		os.Remove(finalPath)
		return WriteResult{}, fmt.Errorf("close blob: %w", err)
	}

	// Register in blobs table with ref_count=0.
	// The INSERT trigger on files/versions will bump it to 1.
	_, err = bs.db.ExecContext(ctx,
		`INSERT INTO blobs (sha256, size, ref_count) VALUES (?, ?, 0)
		 ON CONFLICT(sha256) DO NOTHING`,
		digest, size,
	)
	if err != nil {
		return WriteResult{}, fmt.Errorf("register blob: %w", err)
	}

	return result, nil
}

// Open returns a decompressing reader for the blob with the given sha256.
// The caller must close the returned ReadCloser.
func (bs *BlobStore) Open(sha256hex string) (io.ReadCloser, error) {
	f, err := os.Open(bs.Path(sha256hex))
	if err != nil {
		return nil, fmt.Errorf("open blob %s: %w", sha256hex[:8], err)
	}
	dec, err := DecompressFrom(f)
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("decompress blob: %w", err)
	}
	// Wrap so closing the decompressor also closes the file.
	return &blobReadCloser{ReadCloser: dec, file: f}, nil
}

// DeleteIfUnreferenced removes the blob file from disk if its ref_count is 0.
// Safe to call after deleting a file or version row — the DB trigger has already
// decremented ref_count before this is called.
func (bs *BlobStore) DeleteIfUnreferenced(ctx context.Context, sha256hex string) error {
	var refCount int
	err := bs.db.QueryRowContext(ctx,
		`SELECT ref_count FROM blobs WHERE sha256 = ?`, sha256hex,
	).Scan(&refCount)
	if err == sql.ErrNoRows {
		return nil // already gone
	}
	if err != nil {
		return fmt.Errorf("check ref count: %w", err)
	}
	if refCount > 0 {
		return nil // still in use
	}

	// Remove from disk and blobs table.
	if err := os.Remove(bs.Path(sha256hex)); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete blob file: %w", err)
	}
	_, err = bs.db.ExecContext(ctx, `DELETE FROM blobs WHERE sha256 = ?`, sha256hex)
	return err
}

// path returns the disk path for a blob by its sha256 hex digest.
// Sharded 2/2 like git: <root>/ab/cd/<full-sha>
func (bs *BlobStore) Path(sha256hex string) string {
	if len(sha256hex) < 4 {
		return filepath.Join(bs.root, sha256hex)
	}
	return filepath.Join(bs.root, sha256hex[0:2], sha256hex[2:4], sha256hex)
}

// blobReadCloser closes both the decompressor and the underlying file.
type blobReadCloser struct {
	io.ReadCloser // zstd decoder
	file          *os.File
}

func (b *blobReadCloser) Close() error {
	err := b.ReadCloser.Close()
	if ferr := b.file.Close(); ferr != nil && err == nil {
		err = ferr
	}
	return err
}
