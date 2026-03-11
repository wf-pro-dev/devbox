package types

type SendRequest struct {
	Targets   []string `json:"targets"`
	Broadcast bool     `json:"broadcast"`
	DestDir   string   `json:"dest_dir"`
}

type SendResult struct {
	Target  string `json:"target"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}
