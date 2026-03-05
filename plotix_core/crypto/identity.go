package crypto

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
)

type Identity struct {
	PeerID     string `json:"peer_id"`
	PrivateKey string `json:"private_key"`
	PublicKey  string `json:"public_key"`
}

func InitIdentity(keyPath string) (*Identity, error) {
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		return GenerateAndSave(keyPath)
	}
	return LoadIdentity(keyPath)
}

func GenerateAndSave(keyPath string) (*Identity, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}

	pubHex := hex.EncodeToString(pub)
	peerID := "hex_" + pubHex[:16]

	ident := &Identity{
		PeerID:     peerID,
		PrivateKey: hex.EncodeToString(priv),
		PublicKey:  pubHex,
	}

	err = saveToFile(keyPath, ident)
	return ident, err
}

func LoadIdentity(keyPath string) (*Identity, error) {
	data, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, err
	}

	var ident Identity
	if err := json.Unmarshal(data, &ident); err != nil {
		return nil, err
	}

	return &ident, nil
}

func saveToFile(keyPath string, ident *Identity) error {
	data, err := json.MarshalIndent(ident, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(keyPath, data, 0600)
}
