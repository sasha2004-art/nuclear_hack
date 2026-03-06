package api

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"plotix_core/models"
	"plotix_core/storage"
	"plotix_core/transport"
)

func (s *Server) handleGetPeers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.state.Mu.RLock()
	peers := s.state.Peers
	s.state.Mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(peers)
}

func (s *Server) handleSendMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.SendMessageReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.state.Mu.RLock()
	ip, exists := s.state.Peers[req.PeerID]
	s.state.Mu.RUnlock()

	if !exists {
		http.Error(w, "Peer not found", http.StatusNotFound)
		return
	}

	parents := s.state.GetLastMsgID(req.PeerID)
	msgID := transport.CalculateHash(req.Message, parents)
	chat := transport.ChatPayload{
		ID:      msgID,
		Parents: parents,
		Content: req.Message,
	}
	if err := transport.SendPacket(s.state, s.Broadcast, req.PeerID, ip, "chat", chat); err != nil {
		http.Error(w, "Failed to send: "+err.Error(), http.StatusInternalServerError)
		return
	}

	entity := storage.MessageEntity{
		ID:        msgID,
		Parents:   parents,
		Sender:    s.state.Identity.PeerID,
		Text:      req.Message,
		Timestamp: time.Now().UnixMilli(),
	}
	storage.SaveMessage(req.PeerID, entity)
	s.state.SetLastMsgID(req.PeerID, msgID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "sent"})
}

func (s *Server) handleAddPeerManual(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		IP string `json:"ip"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.state.UpdatePeer("manual_entry_"+req.IP, req.IP)

	json.NewEncoder(w).Encode(map[string]string{"status": "peer_added_locally"})
}

func (s *Server) handleGetHistory(w http.ResponseWriter, r *http.Request) {
	peerID := r.URL.Query().Get("peer_id")
	if peerID == "" {
		http.Error(w, "peer_id required", http.StatusBadRequest)
		return
	}

	history, err := storage.GetHistory(peerID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if history == nil {
		history = []storage.MessageEntity{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(history)
}

func (s *Server) handleWSEvents(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("[WS] Upgrade error:", err)
		return
	}

	s.mu.Lock()
	s.clients[conn] = true
	s.mu.Unlock()

	log.Println("[WS] UI client connected")

	conn.WriteJSON(models.WSEvent{
		Type:    "system_info",
		Payload: "Connected to Plotix Broadcast System",
	})

	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			s.mu.Lock()
			delete(s.clients, conn)
			s.mu.Unlock()
			conn.Close()
			log.Println("[WS] UI client disconnected")
			break
		}
	}
}
