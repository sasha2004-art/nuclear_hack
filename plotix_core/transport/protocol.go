package transport

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

type Packet struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type HandshakePayload struct {
	PeerID       string `json:"peer_id"`
	PublicKey    string `json:"public_key"`
	Name         string `json:"name,omitempty"`
	EphemeralPub string `json:"ephemeral_pub"` // Для X25519
}

type ChatPayload struct {
	ID        string   `json:"id"`
	Parents   []string `json:"parents"`
	Content   string   `json:"content"`   // Здесь теперь будет зашифрованный HEX текст
	Nonce     string   `json:"nonce"`     // HEX Nonce для AES-GCM
	Signature string   `json:"signature"` // Цифровая подпись отправителя
	SenderID  string   `json:"sender_id"`
	TargetID  string   `json:"target_id"`
	Timestamp int64    `json:"timestamp"`
}

type SyncRequestPayload struct {
	PeerID string   `json:"peer_id"`
	Heads  []string `json:"heads"`
}

type SyncResponsePayload struct {
	PeerID   string        `json:"peer_id"`
	Messages []ChatPayload `json:"messages"` // Заменили на ChatPayload для шифрования
}

func CalculateHash(content string, parents []string) string {
	h := sha256.New()
	h.Write([]byte(content))
	for _, p := range parents {
		h.Write([]byte(p))
	}
	return hex.EncodeToString(h.Sum(nil))
}
