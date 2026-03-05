package core

import "sync"

type NodeState struct {
	Mu    sync.RWMutex
	Peers map[string]string
}

func NewNodeState() *NodeState {
	return &NodeState{
		Peers: make(map[string]string),
	}
}
