package api

import (
	"encoding/json"
	"log"
	"net/http"

	"plotix_core/models"
)

func (s *Server) handleGetPeers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")

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
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.SendMessageReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("[CORE] Имитация отправки сообщения юзеру %s: %s", req.PeerID, req.Message)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) handleWSEvents(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("[WS] Ошибка апгрейда соединения:", err)
		return
	}
	defer conn.Close()

	log.Println("[WS] UI Клиент успешно подключился по WebSocket")

	evt := models.WSEvent{
		Type:    "system_info",
		Payload: "Connected to Plotix Core (Stage 1)",
	}
	conn.WriteJSON(evt)

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			log.Println("[WS] Клиент отключился:", err)
			break
		}
	}
}
