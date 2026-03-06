package crypto

import (
	"encoding/hex"
	"testing"
)

func TestE2EESharedSecret(t *testing.T) {
	privA, pubA, _ := GenerateEphemeralKeys()
	privB, pubB, _ := GenerateEphemeralKeys()

	secretA := ComputeSharedSecret(privA, pubB)
	secretB := ComputeSharedSecret(privB, pubA)

	if hex.EncodeToString(secretA) != hex.EncodeToString(secretB) {
		t.Fatal("Shared secrets do not match!")
	}
}

func TestE2EEEncryption(t *testing.T) {
	key := make([]byte, 32) // Dummy key
	plaintext := []byte("Top Secret Message")

	ctxt, nonce, err := EncryptAES(key, plaintext)
	if err != nil {
		t.Fatalf("Encryption failed: %v", err)
	}

	decrypted, err := DecryptAES(key, nonce, ctxt)
	if err != nil {
		t.Fatalf("Decryption failed: %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Fatalf("Decrypted text does not match original")
	}
}
