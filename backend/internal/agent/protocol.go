package agent

type Event struct {
	Type     string `json:"type"`
	Message  string `json:"message,omitempty"`
	Progress int    `json:"progress,omitempty"`
	Success  bool   `json:"success,omitempty"`
	Error    string `json:"error,omitempty"`
}
