package main

import (
	"log"

	"plotix_core/api"
	"plotix_core/core"
	"plotix_core/crypto"
)

func main() {
	log.Println("[BOOT] Запуск Plotix Core (Этап 2: Идентификация)")

	ident, err := crypto.InitIdentity("keystore.json")
	if err != nil {
		log.Fatalf("[FATAL] Ошибка инициализации ключей: %v", err)
	}

	log.Printf("[BOOT] Мой PeerID: %s", ident.PeerID)

	state := core.NewNodeState(ident)

	state.UpdatePeer("hex_deadbeef123456", "192.168.0.10")
	state.UpdatePeer("hex_deadbeef123456", "192.168.0.55")

	server := api.NewServer(state)
	server.Start("8080")
}
