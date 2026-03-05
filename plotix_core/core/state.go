package core

import (
	"sync"

	"plotix_core/crypto"
)

type NodeState struct {
	Mu       sync.RWMutex
	Identity *crypto.Identity
	Peers    map[string]string
}

func NewNodeState(ident *crypto.Identity) *NodeState {
	return &NodeState{
		Identity: ident,
		Peers:    make(map[string]string),
	}
}

func (s *NodeState) UpdatePeer(peerID, ip string) {
	s.Mu.Lock()
	defer s.Mu.Unlock()
	s.Peers[peerID] = ip
}
