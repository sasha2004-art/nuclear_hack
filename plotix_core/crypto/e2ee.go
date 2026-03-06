package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"io"

	"golang.org/x/crypto/curve25519"
)

// GenerateEphemeralKeys создает пару X25519 ключей для сессии
func GenerateEphemeralKeys() (priv []byte, pub []byte, err error) {
	var privateKey [32]byte
	if _, err := io.ReadFull(rand.Reader, privateKey[:]); err != nil {
		return nil, nil, err
	}
	publicKey, err := curve25519.X25519(privateKey[:], curve25519.Basepoint)
	if err != nil {
		return nil, nil, err
	}
	return privateKey[:], publicKey, nil
}

// ComputeSharedSecret вычисляет AES-ключ из своего X25519-приватного и чужого X25519-публичного ключа
func ComputeSharedSecret(myPriv, peerPub []byte) []byte {
	sharedSecret, err := curve25519.X25519(myPriv, peerPub)
	if err != nil {
		return nil
	}
	// Хэшируем секрет для получения надежного 32-байтного ключа AES-256
	hash := sha256.Sum256(sharedSecret)
	return hash[:]
}

// EncryptAES шифрует текст с использованием AES-256-GCM
func EncryptAES(key, plaintext []byte) (ciphertext, nonce []byte, err error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, err
	}

	nonce = make([]byte, aesGCM.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, err
	}

	ciphertext = aesGCM.Seal(nil, nonce, plaintext, nil)
	return ciphertext, nonce, nil
}

// DecryptAES расшифровывает текст с использованием AES-256-GCM
func DecryptAES(key, nonce, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

// SignMessage подписывает ID сообщения (хэш) долгосрочным ключом Ed25519
func SignMessage(privKeyHex, messageID string) string {
	privBytes, err := hex.DecodeString(privKeyHex)
	if err != nil || len(privBytes) != ed25519.PrivateKeySize {
		return ""
	}
	sig := ed25519.Sign(ed25519.PrivateKey(privBytes), []byte(messageID))
	return hex.EncodeToString(sig)
}

// VerifySignature проверяет подпись сообщения
func VerifySignature(pubKeyHex, messageID, sigHex string) bool {
	pubBytes, err := hex.DecodeString(pubKeyHex)
	if err != nil || len(pubBytes) != ed25519.PublicKeySize {
		return false
	}
	sigBytes, err := hex.DecodeString(sigHex)
	if err != nil {
		return false
	}
	return ed25519.Verify(ed25519.PublicKey(pubBytes), []byte(messageID), sigBytes)
}
