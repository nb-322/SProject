package protocol

type UserInfo struct {
	Username string `json:"username,omitempty"`
	Hostname string `json:"hostname,omitempty"`
	MacAddr  string `json:"mac_addr,omitempty"`
	Time     string `json:"time,omitempty"`
}

type Message struct {
	Command  string   `json:"command,omitempty"`
	Response string   `json:"response,omitempty"`
	UserInfo UserInfo `json:"user_info,omitempty"`
}
