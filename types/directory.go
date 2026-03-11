package types

import "github.com/wf-pro-dev/devbox/internal/db"

type Directory struct {
	Prefix    string    `json:"prefix"`
	FileCount int       `json:"file_count"`
	Tags      []string  `json:"tags"`
	Files     []db.File `json:"files,omitempty"`
}
