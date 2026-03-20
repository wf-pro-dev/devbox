package storage

import (
	"fmt"
	"io"

	"github.com/klauspost/compress/zstd"
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
