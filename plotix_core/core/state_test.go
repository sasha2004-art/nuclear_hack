package core

import (
	"testing"

	"plotix_core/crypto"
)

func TestUpdatePeer(t *testing.T) {
	state := NewNodeState(&crypto.Identity{PeerID: "test"})

	state.UpdatePeer("peer_1", "192.168.0.10")

	state.Mu.RLock()
	ip := state.Peers["peer_1"]
	state.Mu.RUnlock()

	if ip != "192.168.0.10" {
		t.Errorf("expected 192.168.0.10, got %s", ip)
	}

	state.UpdatePeer("peer_1", "10.0.0.5")

	state.Mu.RLock()
	ip = state.Peers["peer_1"]
	state.Mu.RUnlock()

	if ip != "10.0.0.5" {
		t.Errorf("expected 10.0.0.5 after update, got %s", ip)
	}
}
