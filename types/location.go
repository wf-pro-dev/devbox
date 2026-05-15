package types

import types "github.com/wf-pro-dev/tailkit/types/integrations"

type Location struct {
	Hostname string
	Paths    []types.PathRule
}

type RemoteFileRequest struct {
	Path string `json:"path"`
}
