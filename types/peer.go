package types

type Peer struct {
	Hostname string `json:"hostname"`
	DNSName  string `json:"dns_name"`
	IP       string `json:"ip"`
	Online   bool   `json:"online"`
}
