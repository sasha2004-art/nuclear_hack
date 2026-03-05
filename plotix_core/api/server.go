package api

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"plotix_core/core"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Server struct {
	state *core.NodeState
}

func NewServer(state *core.NodeState) *Server {
	return &Server{state: state}
}

func (s *Server) Start(port string) {
	http.HandleFunc("/peers", s.handleGetPeers)
	http.HandleFunc("/send_message", s.handleSendMessage)
	http.HandleFunc("/events", s.handleWSEvents)

	addr := fmt.Sprintf("127.0.0.1:%s", port)
	log.Printf("[SERVER] Локальное API запущено на http://%s", addr)

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("[SERVER] Ошибка запуска: %v", err)
	}
}
