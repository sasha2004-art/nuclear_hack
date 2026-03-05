package discovery

import (
	"encoding/json"
	"net"
	"testing"
	"time"

	"plotix_core/core"
	"plotix_core/crypto"
)

func TestAnnounceMsgSerialization(t *testing.T) {
	msg := AnnounceMsg{PeerID: "hex_abc123"}
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded AnnounceMsg
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if decoded.PeerID != "hex_abc123" {
		t.Errorf("expected hex_abc123, got %s", decoded.PeerID)
	}
}

func TestProcessAnnounce(t *testing.T) {
	ident := &crypto.Identity{PeerID: "hex_my_node"}
	state := core.NewNodeState(ident)

	foreignPeerID := "hex_other_node"
	foreignIP := "192.168.1.42"

	state.UpdatePeer(foreignPeerID, foreignIP)

	state.Mu.RLock()
	ip, exists := state.Peers[foreignPeerID]
	state.Mu.RUnlock()

	if !exists {
		t.Error("Peer was not added")
	}
	if ip != foreignIP {
		t.Errorf("expected %s, got %s", foreignIP, ip)
	}

	newIP := "10.0.0.5"
	state.UpdatePeer(foreignPeerID, newIP)

	state.Mu.RLock()
	ip = state.Peers[foreignPeerID]
	state.Mu.RUnlock()

	if ip != newIP {
		t.Errorf("expected updated IP %s, got %s", newIP, ip)
	}
}

func TestSelfAnnounceIgnored(t *testing.T) {
	ident := &crypto.Identity{PeerID: "hex_self"}
	state := core.NewNodeState(ident)

	msg := AnnounceMsg{PeerID: "hex_self"}
	if msg.PeerID != state.Identity.PeerID {
		t.Error("Self-announce filter broken")
	}

	msg2 := AnnounceMsg{PeerID: "hex_other"}
	if msg2.PeerID == state.Identity.PeerID {
		t.Error("Foreign peer incorrectly filtered as self")
	}
}

func TestDiscoveryUDP(t *testing.T) {
	testDiscoveryViaUnicast(t)
}

func testDiscoveryViaUnicast(t *testing.T) {
	ident := &crypto.Identity{PeerID: "hex_my_test_id"}
	state := core.NewNodeState(ident)

	listenAddr, err := net.ResolveUDPAddr("udp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	conn, err := net.ListenUDP("udp4", listenAddr)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	actualAddr := conn.LocalAddr().(*net.UDPAddr)

	go func() {
		buffer := make([]byte, 1024)
		for {
			n, src, err := conn.ReadFromUDP(buffer)
			if err != nil {
				return
			}
			var msg AnnounceMsg
			if err := json.Unmarshal(buffer[:n], &msg); err != nil {
				continue
			}
			if msg.PeerID == state.Identity.PeerID {
				continue
			}
			peerIP := src.IP.String()
			state.UpdatePeer(msg.PeerID, peerIP)
		}
	}()

	time.Sleep(50 * time.Millisecond)

	sendConn, err := net.DialUDP("udp4", nil, actualAddr)
	if err != nil {
		t.Fatal(err)
	}
	defer sendConn.Close()

	fakePeerID := "hex_fake_peer_999"
	msg := AnnounceMsg{PeerID: fakePeerID}
	data, _ := json.Marshal(msg)
	sendConn.Write(data)

	time.Sleep(200 * time.Millisecond)

	state.Mu.RLock()
	ip, exists := state.Peers[fakePeerID]
	state.Mu.RUnlock()

	if !exists {
		t.Error("Peer was not added after receiving UDP packet")
	}
	if ip == "" {
		t.Error("Peer IP is empty")
	}
}
