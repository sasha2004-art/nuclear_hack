package core

import (
	"sync"

	"plotix_core/crypto"
)

type NodeState struct {
	Mu       sync.RWMutex
	Identity *crypto.Identity
	Peers    map[string]string

	NewPeerChan chan string
}

func NewNodeState(ident *crypto.Identity) *NodeState {
	return &NodeState{
		Identity:    ident,
		Peers:       make(map[string]string),
		NewPeerChan: make(chan string, 10),
	}
}

func (s *NodeState) UpdatePeer(peerID, ip string) {
	s.Mu.Lock()
	_, exists := s.Peers[peerID]
	s.Peers[peerID] = ip
	s.Mu.Unlock()

	if !exists {
		s.NewPeerChan <- ip
	}
}
