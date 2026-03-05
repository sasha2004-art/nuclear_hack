package main

import (
	"log"

	"plotix_core/api"
	"plotix_core/core"
)

func main() {
	log.Println("Запуск Plotix Core (Этап 1: Фундамент)...")

	state := core.NewNodeState()

	state.Mu.Lock()
	state.Peers["test_peer_123"] = "192.168.0.15"
	state.Mu.Unlock()

	server := api.NewServer(state)
	server.Start("8080")
}
