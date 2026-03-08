package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// BlobStore manages raw file content on disk.
type BlobStore struct {
	root string
}

// NewBlobStore creates a BlobStore rooted at the given directory.
func NewBlobStore(root string) (*BlobStore, error) {
	if err := os.MkdirAll(root, 0755); err != nil {
		return nil, fmt.Errorf("create blob dir: %w", err)
	}
	return &BlobStore{root: root}, nil
}

// Write saves the contents of r to disk under fileID and returns the
// file size and SHA256 hex digest. The file is written atomically via
// a temp file to avoid partial writes being visible.
func (bs *BlobStore) Write(fileID string, r io.Reader) (size int64, sha256hex string, err error) {
	// Write to a temp file first.
	tmp, err := os.CreateTemp(bs.root, "upload-*")
	if err != nil {
		return 0, "", fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() {
		// Clean up temp file on any error.
		if err != nil {
			os.Remove(tmpPath)
		}
	}()

	h := sha256.New()
	mw := io.MultiWriter(tmp, h)

	size, err = io.Copy(mw, r)
	if err != nil {
		tmp.Close()
		return 0, "", fmt.Errorf("write blob: %w", err)
	}
	if err = tmp.Close(); err != nil {
		return 0, "", fmt.Errorf("close temp file: %w", err)
	}

	sha256hex = hex.EncodeToString(h.Sum(nil))

	// Atomically move to the final path.
	finalPath := bs.path(fileID)
	if err = os.MkdirAll(filepath.Dir(finalPath), 0755); err != nil {
		return 0, "", fmt.Errorf("create blob subdir: %w", err)
	}
	if err = os.Rename(tmpPath, finalPath); err != nil {
		return 0, "", fmt.Errorf("move blob: %w", err)
	}

	return size, sha256hex, nil
}

// Read opens the blob for the given fileID for reading.
// The caller is responsible for closing the returned file.
func (bs *BlobStore) Read(fileID string) (*os.File, error) {
	f, err := os.Open(bs.path(fileID))
	if err != nil {
		return nil, fmt.Errorf("open blob %s: %w", fileID, err)
	}
	return f, nil
}

// Delete removes the blob for the given fileID from disk.
func (bs *BlobStore) Delete(fileID string) error {
	if err := os.Remove(bs.path(fileID)); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete blob %s: %w", fileID, err)
	}
	return nil
}

// path returns the full disk path for a given fileID.
// Files are sharded into subdirectories using the first 2 chars of the ID
// to avoid having too many files in a single directory.
func (bs *BlobStore) path(fileID string) string {
	if len(fileID) < 2 {
		return filepath.Join(bs.root, fileID)
	}
	return filepath.Join(bs.root, fileID[:2], fileID)
}

// BlobPath returns the full disk path for a given fileID.
// Exposed so callers can store the path in the database.
func (bs *BlobStore) BlobPath(fileID string) string {
	return bs.path(fileID)
}
