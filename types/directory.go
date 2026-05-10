package types

// DirEntry is one item in a virtual-directory listing.
// IsDir distinguishes a virtual sub-directory (collapsed CommonPrefix) from a
// concrete file.
type DirEntry struct {
	// Name is the bare segment — no leading or trailing slashes.
	// "config" for a sub-directory, "nginx.conf" for a file.
	Name string `json:"name"`

	// IsDir is true when this entry represents a virtual sub-directory.
	IsDir bool `json:"is_dir"`

	// Prefix is set only for directory entries: "/myapp/config/".
	// Empty string for file entries.
	Prefix string `json:"prefix,omitempty"`

	// FileCount is the total number of files under Prefix.
	// One for file entries.
	FileCount int `json:"file_count,omitempty"`

	// File is set only for file entries.
	// Nil for directory entries.
	File *File `json:"file,omitempty"`
}

// DirListing is the response body for GET /dirs and GET /dirs/{dir}.
// A single shape serves both the CLI (JSON decode into tabwriter output) and
// the web frontend (column-view rendering).
type DirListing struct {
	// Prefix is the canonical directory prefix that was listed: "/myapp/".
	Prefix string `json:"prefix"`

	// Tags are the union of tags across all files directly under Prefix.
	Tags []string `json:"tags,omitempty"`

	// Entries holds the direct children in the order returned by the DB
	// (directories first, then files, both alphabetical).
	Entries []DirEntry `json:"entries"`
}
