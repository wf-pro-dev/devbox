package models

import (
	"errors"
	"path"
	"strings"

	"github.com/wf-pro-dev/devbox/internal/db"
	"github.com/wf-pro-dev/devbox/types"
)

type dirAccumulator struct {
	fileCount        int
	totalSize        int64
	latestUpdatedAt  string
	oldestCreatedAt  string
	oldestUploadedBy string
}

func newerTimestamp(a, b string) string {
	if a == "" {
		return b
	}
	if b == "" {
		return a
	}
	if a >= b {
		return a
	}
	return b
}

func olderTimestamp(a, b string) string {
	if a == "" {
		return b
	}
	if b == "" {
		return a
	}
	if a <= b {
		return a
	}
	return b
}

// ── Sentinel errors ────────────────────────────────────────────────────────

var (
	ErrEmpty         = errors.New("path must not be empty")
	ErrNotAbsolute   = errors.New("path must be absolute (start with /)")
	ErrDotSegment    = errors.New("path must not contain . or .. segments after cleaning")
	ErrTrailingSlash = errors.New("file path must not end with /")
)

// ── Write-path helpers ─────────────────────────────────────────────────────

// CanonicalDir returns the canonical directory prefix for s.
//
//	"nginx"          → "/nginx/"
//	"myapp/config"   → "/myapp/config/"
//	"/myapp//config" → "/myapp/config/"
//	"/myapp/config/" → "/myapp/config/"
//
// Returns an error if s is empty or cannot be made safe.
func CanonicalDir(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", ErrEmpty
	}
	// Ensure absolute before cleaning so path.Clean("/"+s) works correctly.
	if !strings.HasPrefix(s, "/") {
		s = "/" + s
	}
	cleaned := path.Clean(s)
	if cleaned == "." || cleaned == "/" {
		return "", ErrEmpty
	}
	result := cleaned + "/"
	return result, nil
}

// MustCanonicalDir is like CanonicalDir but panics on error.
// Use only in tests or package-level var initialisations.
func MustCanonicalDir(s string) string {
	p, err := CanonicalDir(s)
	if err != nil {
		panic("pathutil.MustCanonicalDir: " + err.Error())
	}
	return p
}

// CanonicalFile returns the canonical file path for s.
//
//	"myapp/config/nginx.conf"   → "/myapp/config/nginx.conf"
//	"/myapp/config/nginx.conf/" → error (trailing slash)
//
// Returns an error if s is empty, ends with "/" or cannot be made safe.
func CanonicalFile(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", ErrEmpty
	}
	if strings.HasSuffix(s, "/") {
		return "", ErrTrailingSlash
	}
	if !strings.HasPrefix(s, "/") {
		s = "/" + s
	}
	cleaned := path.Clean(s)
	if cleaned == "/" {
		return "", ErrEmpty
	}
	return cleaned, nil
}

// ValidateDir returns a non-nil error if p is not a valid canonical directory
// prefix (absolute, ends with "/", no dot segments).
func ValidateDir(p string) error {
	if p == "" {
		return ErrEmpty
	}
	if !strings.HasPrefix(p, "/") {
		return ErrNotAbsolute
	}
	if !strings.HasSuffix(p, "/") {
		// Tolerate missing trailing slash — callers should normalise before
		// storing, but validation should be strict.
		return errors.New("directory prefix must end with /")
	}
	if strings.Contains(p, "//") {
		return errors.New("path must not contain double slashes")
	}
	for _, seg := range Segments(p) {
		if seg == "." || seg == ".." {
			return ErrDotSegment
		}
	}
	return nil
}

// ── Read-path helpers ──────────────────────────────────────────────────────

// Segments splits a canonical path into its non-empty segments.
//
//	"/myapp/config/nginx.conf" → ["myapp", "config", "nginx.conf"]
//	"/myapp/config/"           → ["myapp", "config"]
func Segments(p string) []string {
	trimmed := strings.Trim(p, "/")
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "/")
}

// Depth returns the number of segments in a canonical path.
//
//	"/"              → 0
//	"/myapp/"        → 1
//	"/myapp/config/" → 2
func Depth(p string) int {
	return len(Segments(p))
}

// Parent returns the canonical directory prefix of p.
//
//	"/myapp/config/nginx.conf" → "/myapp/config/"
//	"/myapp/config/"           → "/myapp/"
//	"/myapp/"                  → "/"
func Parent(p string) string {
	// Strip trailing slash to let path.Dir work correctly.
	cleaned := strings.TrimSuffix(p, "/")
	parent := path.Dir(cleaned)
	if parent == "." {
		return "/"
	}
	if !strings.HasSuffix(parent, "/") {
		parent += "/"
	}
	return parent
}

// IsDirectChild reports whether p is a direct child of prefix.
// Works for both files and subdirectories.
//
//	IsDirectChild("/myapp/nginx.conf",      "/myapp/")        → true
//	IsDirectChild("/myapp/config/nginx.conf","/myapp/")       → false  (nested)
//	IsDirectChild("/myapp/config/",          "/myapp/")       → true   (sub-dir)
func IsDirectChild(p, prefix string) bool {
	if !strings.HasPrefix(p, prefix) {
		return false
	}
	rel := p[len(prefix):]
	if rel == "" {
		return false
	}
	// A direct child has at most one segment.
	// For directories the rel is "seg/"; for files it is "filename".
	rel = strings.TrimSuffix(rel, "/")
	return !strings.Contains(rel, "/")
}

// Join joins a canonical prefix with a relative path, returning a canonical
// file path. Useful when assembling full paths from form uploads.
//
//	Join("/myapp/", "config/nginx.conf") → "/myapp/config/nginx.conf"
func Join(prefix, rel string) string {
	rel = strings.TrimPrefix(rel, "/")
	return path.Clean(prefix + rel)
}

// PathOf is a function that extracts the canonical path from a file record F.
// Callers supply this to decouple the algorithm from the concrete db.File type.
type PathOf[F any] func(F) string

// ListDirect returns the direct children of prefix from a pre-sorted slice of
// files, using the CommonPrefix algorithm (identical to S3 ListObjectsV2 with
// Delimiter="/").
//
// Complexity: O(n) time, O(k) space where k is the number of distinct direct
// children (files + sub-directories).
//
// Requirements:
//   - files must be sorted lexicographically by path (natural for a DB
//     query with ORDER BY path).
//   - Every path in files must be canonical (use CanonicalFile before storing).
//   - prefix must be a canonical directory prefix (use CanonicalDir).
//
// The algorithm:
//  1. Strip the prefix from each path to get a relative path.
//  2. If the relative path contains no "/", it is a direct file child → emit.
//  3. If it contains a "/", everything up to and including the first "/" is a
//     CommonPrefix (virtual sub-directory) → accumulate counts, emit once when
//     the prefix changes.
func ListDirect(files []db.File, prefix string, pathOf PathOf[db.File]) []types.DirEntry {
	// Normalise prefix: must end with "/" and start with "/".
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	if !strings.HasPrefix(prefix, "/") {
		prefix = "/" + prefix
	}

	var (
		result      []types.DirEntry
		lastSubdir  string // tracks the current common-prefix accumulation
		subdirStats = map[string]*dirAccumulator{}
	)

	flush := func() {
		if lastSubdir == "" {
			return
		}
		stats := subdirStats[lastSubdir]
		// Segment name without slashes for display.
		seg := strings.Trim(lastSubdir[len(prefix):], "/")
		entry := types.DirEntry{
			Name:   seg,
			Prefix: lastSubdir,
			IsDir:  true,
		}
		if stats != nil {
			entry.FileCount = stats.fileCount
			entry.Stats = &types.DirectoryStats{
				TotalSize:        stats.totalSize,
				LatestUpdatedAt:  stats.latestUpdatedAt,
				OldestCreatedAt:  stats.oldestCreatedAt,
				OldestUploadedBy: stats.oldestUploadedBy,
			}
		}
		result = append(result, entry)
		lastSubdir = ""
	}

	for _, f := range files {
		p := pathOf(f)
		if !strings.HasPrefix(p, prefix) {
			continue
		}
		rel := p[len(prefix):]
		if rel == "" {
			continue
		}

		slashIdx := strings.Index(rel, "/")
		if slashIdx < 0 {
			// Direct file child — flush any pending sub-directory first.
			flush()
			result = append(result, types.DirEntry{
				Name:      rel,
				IsDir:     false,
				File:      &types.File{File: f},
				FileCount: 1,
			})
		} else {
			// The file lives inside a sub-directory.
			// The sub-directory prefix is everything up to and including the "/".
			subPrefix := prefix + rel[:slashIdx+1]
			stats := subdirStats[subPrefix]
			if stats == nil {
				stats = &dirAccumulator{}
				subdirStats[subPrefix] = stats
			}
			stats.fileCount++
			stats.totalSize += f.Size
			updatedAt := f.UpdatedAt
			if updatedAt == "" {
				updatedAt = f.CreatedAt
			}
			stats.latestUpdatedAt = newerTimestamp(stats.latestUpdatedAt, updatedAt)
			prevOldest := stats.oldestCreatedAt
			stats.oldestCreatedAt = olderTimestamp(stats.oldestCreatedAt, f.CreatedAt)
			if prevOldest == "" || stats.oldestCreatedAt != prevOldest {
				stats.oldestUploadedBy = f.UploadedBy
			}
			if subPrefix == lastSubdir {
				// Same sub-directory as the previous file — stats already updated.
			} else {
				// New sub-directory encountered — flush the previous one.
				flush()
				lastSubdir = subPrefix
			}
		}
	}
	flush() // emit the final pending sub-directory if any

	return result
}

// ── SQL query for DB-side directory listing ────────────────────────────────
//
// If your vault is large (> 50 k files) prefer having the DB resolve virtual
// directories instead of loading all rows into Go.
//
// Add the following to your sqlc .sql file.  It uses a slash-count filter to
// return only direct children of a given prefix, then groups sub-directory
// entries so you get one row per virtual sub-directory.
//
// The query is written for PostgreSQL / SQLite (both support the same syntax).
//
// ─── sqlc query (add to your .sql definitions) ────────────────────────────
//
//   -- name: ListDirectChildren :many
//   --
//   -- Returns direct children (files and virtual sub-directories) of :prefix.
//   -- :prefix must be a canonical directory prefix ending in "/", e.g. "/myapp/".
//   -- :prefix_depth is the number of slash-separated segments in :prefix, e.g.
//   --   "/"        → 0
//   --   "/myapp/"  → 1
//   -- This is used so the query can count slashes inline and filter to depth+1.
//   --
//   -- Returned columns:
//   --   path       TEXT  — for files: the full path.
//   --                      for dirs:  the virtual prefix ending in "/".
//   --   is_dir     BOOL  — true when the row represents a virtual sub-directory.
//   --   file_count INT   — 1 for file rows; total files under the sub-prefix for dir rows.
//   --
//   SELECT
//     CASE
//       -- File is a direct child: depth equals prefix_depth + 1 (one more slash).
//       WHEN (LENGTH(path) - LENGTH(REPLACE(path, '/', ''))) = sqlc.arg(prefix_depth) + 1
//         THEN path
//       -- File lives deeper: truncate to the first sub-directory segment.
//       ELSE SUBSTR(path, 1,
//              INSTR(SUBSTR(path, LENGTH(sqlc.arg(prefix)) + 1), '/') + LENGTH(sqlc.arg(prefix)))
//     END AS path,
//     (LENGTH(path) - LENGTH(REPLACE(path, '/', ''))) > sqlc.arg(prefix_depth) + 1 AS is_dir,
//     COUNT(*) AS file_count
//   FROM files
//   WHERE path LIKE sqlc.arg(prefix) || '%'
//   GROUP BY 1, 2
//   ORDER BY 2 DESC, 1 ASC   -- directories first, then files, both alphabetical
//   ;
//
// ── SQLite note ───────────────────────────────────────────────────────────
//   SQLite uses INSTR / SUBSTR (shown above).
//   PostgreSQL equivalent for the SUBSTR+INSTR:
//     LEFT(path, LENGTH(prefix) + POSITION('/' IN SUBSTRING(path FROM LENGTH(prefix)+1)))
//
// ── Index recommendation ──────────────────────────────────────────────────
//   CREATE INDEX idx_files_path ON files (path);
//   A B-tree index on path makes the LIKE 'prefix%' prefix-scan O(log n + k)
//   instead of O(n).  The slash-count arithmetic is evaluated only on the
//   matching rows, keeping the query fast even for large tables.
