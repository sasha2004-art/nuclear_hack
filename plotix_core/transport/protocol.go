package transport

import "encoding/json"

type Packet struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type HandshakePayload struct {
	PeerID    string `json:"peer_id"`
	PublicKey string `json:"public_key"`
}

type ChatPayload struct {
	Content string `json:"content"`
}
