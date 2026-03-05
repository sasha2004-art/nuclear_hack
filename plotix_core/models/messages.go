package models

type SendMessageReq struct {
	PeerID  string `json:"peer_id"`
	Message string `json:"message"`
}

type Peer struct {
	ID string `json:"id"`
	IP string `json:"ip"`
}

type WSEvent struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}
