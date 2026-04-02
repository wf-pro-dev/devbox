package storage

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/klauspost/compress/zstd"
	"github.com/wf-pro-dev/devbox/internal/db"
)

// zstdEncoder is a package-level encoder reused across writes (thread-safe).
var zstdEncoder, _ = zstd.NewWriter(nil,
	zstd.WithEncoderLevel(zstd.SpeedDefault),
)

// zstdDecoder is a package-level decoder reused across reads (thread-safe).
var zstdDecoder, _ = zstd.NewReader(nil)

// CompressTo compresses src into dst using zstd and returns bytes written.
func CompressTo(dst io.Writer, src io.Reader) (int64, error) {
	enc, err := zstd.NewWriter(dst, zstd.WithEncoderLevel(zstd.SpeedDefault))
	if err != nil {
		return 0, fmt.Errorf("zstd writer: %w", err)
	}
	n, err := io.Copy(enc, src)
	if err != nil {
		enc.Close()
		return 0, fmt.Errorf("compress: %w", err)
	}
	if err := enc.Close(); err != nil {
		return 0, fmt.Errorf("flush zstd: %w", err)
	}
	return n, nil
}

// DecompressFrom wraps src in a zstd decoder and returns it as an io.ReadCloser.
// The caller must close the returned reader.
func DecompressFrom(src io.Reader) (io.ReadCloser, error) {
	dec, err := zstd.NewReader(src)
	if err != nil {
		return nil, fmt.Errorf("zstd reader: %w", err)
	}
	return dec.IOReadCloser(), nil
}

// readBlob opens the zstd-compressed blob at blobPath, decompresses it, and
// returns the raw bytes. The BlobStore stores all blobs zstd-compressed but
// tailkitd expects raw file content.
func ReadBlob(blobPath string) ([]byte, error) {
	f, err := os.Open(blobPath)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", blobPath, err)
	}
	defer f.Close()

	rc, err := DecompressFrom(f)
	if err != nil {
		return nil, fmt.Errorf("decompress %s: %w", blobPath, err)
	}
	defer rc.Close()

	return io.ReadAll(rc)
}

func CreateTarball(dirName string, files []db.File, blobs *BlobStore) (*os.File, error) {

	tarballFilePath := filepath.Join(os.TempDir(), fmt.Sprintf("%s.tar.gz", dirName))

	file, err := os.Create(tarballFilePath)
	if err != nil {
		return nil, fmt.Errorf("Could not create tarball file '%s', got error '%s'", tarballFilePath, err)
	}

	gzipWriter := gzip.NewWriter(file)

	tarWriter := tar.NewWriter(gzipWriter)

	for _, file := range files {

		blobPath := blobs.Path(file.Sha256)
		blob, err := ReadBlob(blobPath)
		if err != nil {
			return nil, fmt.Errorf("Could not read blob '%s', got error '%s'", file.Sha256, err.Error())
		}

		TEMP_DIR := os.TempDir()
		tmp, err := os.CreateTemp(TEMP_DIR, ".tailkitd-recv-*")
		if err != nil {
			return nil, fmt.Errorf("Could not create temp file, got error '%s'", err.Error())
		}
		tmpPath := tmp.Name()
		defer os.Remove(tmpPath) // no-op after successful rename

		_, err = io.Copy(tmp, bytes.NewReader(blob))
		if err != nil {
			return nil, fmt.Errorf("Could not copy blob to temp file, got error '%s'", err.Error())
		}

		err = addFileToTarWriter(tmpPath, strings.TrimPrefix(file.Path, "/"), tarWriter)
		if err != nil {
			return nil, fmt.Errorf("Could not add file '%s', to tarball, got error '%s'", file.Path, err.Error())
		}

	}

	// Close in order: tar → gzip → (caller closes the file)
	if err := tarWriter.Close(); err != nil {
		gzipWriter.Close()
		file.Close()
		return nil, fmt.Errorf("close tar writer: %w", err)
	}
	if err := gzipWriter.Close(); err != nil {
		file.Close()
		return nil, fmt.Errorf("close gzip writer: %w", err)
	}

	// Seek back to the start so the caller can read/stat a complete file
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		file.Close()
		return nil, fmt.Errorf("seek tarball: %w", err)
	}

	return file, nil
}

// Private methods

func addFileToTarWriter(readPath, filePath string, tarWriter *tar.Writer) error {
	file, err := os.Open(readPath)
	if err != nil {
		return fmt.Errorf("Could not open file '%s', got error '%s'", readPath, err.Error())
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("Could not get stat for file '%s', got error '%s'", readPath, err.Error())
	}

	header := &tar.Header{
		Name:    filePath,
		Size:    stat.Size(),
		Mode:    int64(stat.Mode()),
		ModTime: stat.ModTime(),
	}

	err = tarWriter.WriteHeader(header)
	if err != nil {
		return fmt.Errorf("Could not write header for file '%s', got error '%s'", filePath, err.Error())
	}

	_, err = io.Copy(tarWriter, file)
	if err != nil {
		return fmt.Errorf("Could not copy the file '%s' data to the tarball, got error '%s'", filePath, err.Error())
	}

	return nil
}
