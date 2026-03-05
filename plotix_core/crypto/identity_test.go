package crypto

import (
	"os"
	"testing"
)

func TestInitIdentity(t *testing.T) {
	testFile := "test_keystore.json"

	os.Remove(testFile)
	defer os.Remove(testFile)

	ident1, err := InitIdentity(testFile)
	if err != nil {
		t.Fatalf("Key generation error: %v", err)
	}

	if ident1.PeerID == "" || len(ident1.PrivateKey) == 0 || len(ident1.PublicKey) == 0 {
		t.Errorf("Empty keys generated: %+v", ident1)
	}

	ident2, err := InitIdentity(testFile)
	if err != nil {
		t.Fatalf("Key loading error: %v", err)
	}

	if ident1.PeerID != ident2.PeerID {
		t.Errorf("PeerID mismatch: expected %s, got %s", ident1.PeerID, ident2.PeerID)
	}

	if ident1.PrivateKey != ident2.PrivateKey {
		t.Error("Private keys do not match after loading from disk")
	}
}
