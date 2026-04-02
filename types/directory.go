package types

import "github.com/wf-pro-dev/devbox/internal/db"

type Directory[T File | db.File] struct {
	Prefix    string   `json:"prefix"`
	FileCount int      `json:"file_count"`
	Tags      []string `json:"tags"`
	Files     []T      `json:"files,omitempty"`
}
