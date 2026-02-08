package client

type IngestResponse struct {
	Status    string `json:"status"`
	ClockSkew bool   `json:"clockSkew"`
}

type BatchResponse struct {
	Status   string   `json:"status"`
	Accepted int      `json:"accepted"`
	Rejected int      `json:"rejected"`
	Errors   []string `json:"errors"`
}

type EnrollResponse struct {
	HostID    string `json:"hostId"`
	HostToken string `json:"hostToken"`
}
