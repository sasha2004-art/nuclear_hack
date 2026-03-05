package core

import (
	"net"
	"sync"
	"time"

	"plotix_core/crypto"
)

var StartTime = time.Now()

type NodeState struct {
	Mu          sync.RWMutex
	Identity    *crypto.Identity
	Peers       map[string]string
	ActiveConns map[string]net.Conn
	NewPeerChan chan string
}

func NewNodeState(ident *crypto.Identity) *NodeState {
	return &NodeState{
		Identity:    ident,
		Peers:       make(map[string]string),
		ActiveConns: make(map[string]net.Conn),
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

func (s *NodeState) SaveConnection(peerID string, conn net.Conn) {
	s.Mu.Lock()
	defer s.Mu.Unlock()
	if oldConn, ok := s.ActiveConns[peerID]; ok {
		oldConn.Close()
	}
	s.ActiveConns[peerID] = conn
}

func (s *NodeState) GetConnection(peerID string) net.Conn {
	s.Mu.RLock()
	defer s.Mu.RUnlock()
	return s.ActiveConns[peerID]
}

func (s *NodeState) RemoveConnection(peerID string) {
	s.Mu.Lock()
	defer s.Mu.Unlock()
	delete(s.ActiveConns, peerID)
}
