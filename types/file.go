package types

import "github.com/wf-pro-dev/devbox/internal/db"

type File struct {
	db.File
	Tags     []string `json:"tags"`
	Source   string   `json:"source,omitempty"`
	Hostname string   `json:"hostname,omitempty"`
}
