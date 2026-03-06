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

// --- Дополнения для Этапа 11 (File Transfer) ---

type FileStartPayload struct {
	TransferID string `json:"transfer_id"`
	FileName   string `json:"file_name"`
	FileSize   int64  `json:"file_size"`
	SenderID   string `json:"sender_id"`
	TargetID   string `json:"target_id"`
}

type FileChunkPayload struct {
	TransferID string `json:"transfer_id"`
	ChunkIndex int    `json:"chunk_index"`
	TotalChunks int   `json:"total_chunks"`
	Data       []byte `json:"data"`      // Зашифрованные байты (Go сам сделает Base64)
	Nonce      []byte `json:"nonce"`     // Nonce в байтах
	Signature  string `json:"signature"` // Подпись чанка
}
