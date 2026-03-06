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
	PeerNames   map[string]string
	PeerAliases map[string]string
	LastSeen    map[string]time.Time
	ActiveConns map[string]net.Conn
	NewPeerChan chan string
	LastMsgIDs  map[string]string
	DisplayName func() string
}

func NewNodeState(ident *crypto.Identity) *NodeState {
	return &NodeState{
		Identity:    ident,
		Peers:       make(map[string]string),
		PeerNames:   make(map[string]string),
		PeerAliases: make(map[string]string),
		LastSeen:    make(map[string]time.Time),
		ActiveConns: make(map[string]net.Conn),
		NewPeerChan: make(chan string, 10),
		LastMsgIDs:  make(map[string]string),
	}
}

func (s *NodeState) UpdateLastSeen(peerID string) {
	s.Mu.Lock()
	defer s.Mu.Unlock()
	s.LastSeen[peerID] = time.Now()
}

func (s *NodeState) IsPeerOnline(peerID string) bool {
	s.Mu.RLock()
	defer s.Mu.RUnlock()
	last, ok := s.LastSeen[peerID]
	if !ok {
		return false
	}
	return time.Since(last) < 12*time.Second
}

func (s *NodeState) GetLastMsgID(peerID string) []string {
	s.Mu.RLock()
	defer s.Mu.RUnlock()
	lastID, ok := s.LastMsgIDs[peerID]
	if !ok {
		return []string{}
	}
	return []string{lastID}
}

func (s *NodeState) SetLastMsgID(peerID, msgID string) {
	s.Mu.Lock()
	defer s.Mu.Unlock()
	s.LastMsgIDs[peerID] = msgID
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

func (s *NodeState) SetPeerName(peerID, name string) {
	s.Mu.Lock()
	defer s.Mu.Unlock()
	if name == "" {
		delete(s.PeerNames, peerID)
	} else {
		s.PeerNames[peerID] = name
	}
}

func (s *NodeState) SetPeerAlias(peerID, name string) {
	s.Mu.Lock()
	defer s.Mu.Unlock()
	if name == "" {
		delete(s.PeerAliases, peerID)
	} else {
		s.PeerAliases[peerID] = name
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

func (s *NodeState) ResetConnections() {
	s.Mu.Lock()
	defer s.Mu.Unlock()
	for id, conn := range s.ActiveConns {
		conn.Close()
		delete(s.ActiveConns, id)
	}
	s.Peers = make(map[string]string)
	s.PeerNames = make(map[string]string)
	s.PeerAliases = make(map[string]string)
	s.LastSeen = make(map[string]time.Time)
	s.LastMsgIDs = make(map[string]string)
}
