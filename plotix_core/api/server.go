package api

import (
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"plotix_core/core"
	"plotix_core/models"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Server struct {
	state     *core.NodeState
	clients   map[*websocket.Conn]bool
	mu        sync.Mutex
	Broadcast chan models.WSEvent
	uiFS      fs.FS
}

func NewServer(state *core.NodeState, uiFS fs.FS) *Server {
	s := &Server{
		state:     state,
		clients:   make(map[*websocket.Conn]bool),
		Broadcast: make(chan models.WSEvent, 100),
		uiFS:      uiFS,
	}
	go s.listenBroadcast()
	return s
}

func (s *Server) listenBroadcast() {
	for event := range s.Broadcast {
		s.mu.Lock()
		for client := range s.clients {
			err := client.WriteJSON(event)
			if err != nil {
				client.Close()
				delete(s.clients, client)
			}
		}
		s.mu.Unlock()
	}
}

func (s *Server) Start(port string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/peers", s.handleGetPeers)
	mux.HandleFunc("/send_message", s.handleSendMessage)
	mux.HandleFunc("/events", s.handleWSEvents)
	mux.HandleFunc("/add_peer", s.handleAddPeerManual)
	mux.HandleFunc("/history", s.handleGetHistory)

	if s.uiFS != nil {
		fileServer := http.FileServer(http.FS(s.uiFS))
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path
			if path == "/" {
				path = "/index.html"
			}
			_, err := fs.Stat(s.uiFS, path[1:])
			if err != nil {
				r.URL.Path = "/"
			}
			fileServer.ServeHTTP(w, r)
		})
	}

	corsHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		mux.ServeHTTP(w, r)
	})

	addr := fmt.Sprintf("0.0.0.0:%s", port)
	log.Printf("[SERVER] API + UI on http://%s", addr)

	if err := http.ListenAndServe(addr, corsHandler); err != nil {
		log.Fatalf("[SERVER] Error: %v", err)
	}
}
