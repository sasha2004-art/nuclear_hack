package api

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"plotix_core/accounts"
	"plotix_core/models"
	"plotix_core/storage"
	"plotix_core/transport"
)

type PeerEntry struct {
	IP     string `json:"ip"`
	Name   string `json:"name,omitempty"`
	Online bool   `json:"online"`
}

func (s *Server) handleGetPeers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	result := make(map[string]PeerEntry)

	s.state.Mu.RLock()
	now := time.Now()
	for id, ip := range s.state.Peers {
		name := s.state.PeerAliases[id]
		if name == "" {
			name = s.state.PeerNames[id]
		}
		isOnline := false
		if last, ok := s.state.LastSeen[id]; ok {
			isOnline = now.Sub(last) < 12*time.Second
		}
		result[id] = PeerEntry{
			IP:     ip,
			Name:   name,
			Online: isOnline,
		}
	}
	s.state.Mu.RUnlock()

	knownPeers, _ := storage.GetKnownPeers()

	s.state.Mu.RLock()
	for _, id := range knownPeers {
		if _, exists := result[id]; !exists {
			name := s.state.PeerAliases[id]
			result[id] = PeerEntry{
				IP:     "",
				Name:   name,
				Online: false,
			}
		}
	}
	s.state.Mu.RUnlock()

	contacts, _ := storage.GetAllContacts()
	for id, name := range contacts {
		if _, exists := result[id]; !exists {
			result[id] = PeerEntry{
				IP:     "",
				Name:   name,
				Online: false,
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
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
	ip, online := s.state.Peers[req.PeerID]
	s.state.Mu.RUnlock()

	parents := s.state.GetLastMsgID(req.PeerID)
	msgID := transport.CalculateHash(req.Message, parents)
	now := time.Now().UnixMilli()

	entity := storage.MessageEntity{
		ID:        msgID,
		Parents:   parents,
		Sender:    s.state.Identity.PeerID,
		Text:      req.Message,
		Timestamp: now,
		Delivered: false,
	}
	storage.SaveMessage(req.PeerID, entity)
	s.state.SetLastMsgID(req.PeerID, msgID)

	if online {
		chat := transport.ChatPayload{
			ID:        msgID,
			Parents:   parents,
			Content:   req.Message,
			SenderID:  s.state.Identity.PeerID,
			TargetID:  req.PeerID,
			Timestamp: now,
		}
		if err := transport.SendPacket(s.state, s.Broadcast, req.PeerID, ip, "chat", chat); err != nil {
			log.Printf("[API] Send failed, queued for retry: %v", err)
		}
	} else {
		log.Printf("[API] Peer %s offline, message saved for later delivery", req.PeerID)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "queued"})
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

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	s.state.Mu.RLock()
	peerID := s.state.Identity.PeerID
	s.state.Mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"peer_id": peerID})
}

func (s *Server) handleAccounts(w http.ResponseWriter, r *http.Request) {
	list := s.AccountMgr.ListAccounts()
	if list == nil {
		list = []accounts.AccountInfo{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"accounts":  list,
		"active_id": s.AccountMgr.ActivePeerID,
	})
}

func (s *Server) handleCreateAccount(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	info, _, err := s.AccountMgr.CreateAccount(req.Name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

func (s *Server) handleSwitchAccount(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		PeerID string `json:"peer_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if s.SwitchAccount == nil {
		http.Error(w, "Switch not configured", http.StatusInternalServerError)
		return
	}

	if err := s.SwitchAccount(req.PeerID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "switched", "peer_id": req.PeerID})
}

func (s *Server) handleRenameAccount(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		PeerID string `json:"peer_id"`
		Name   string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if !s.AccountMgr.SetName(req.PeerID, req.Name) {
		http.Error(w, "Account not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "renamed"})
}

func (s *Server) handleSetGhost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		PeerID string `json:"peer_id"`
		Ghost  bool   `json:"ghost"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if !s.AccountMgr.SetGhost(req.PeerID, req.Ghost) {
		http.Error(w, "Account not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok", "ghost": req.Ghost})
}

func (s *Server) handleRenamePeer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		PeerID string `json:"peer_id"`
		Name   string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := storage.SaveContact(req.PeerID, req.Name); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.state.SetPeerAlias(req.PeerID, req.Name)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
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

func (s *Server) handleSendFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	peerID := r.URL.Query().Get("peer_id")
	if peerID == "" {
		http.Error(w, "peer_id required", http.StatusBadRequest)
		return
	}

	// 32MB в памяти, остальное сбрасывается во временные файлы ОС (поддержка до 1 ГБ)
	r.ParseMultipartForm(32 << 20)
	file, handler, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error retrieving file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	s.state.Mu.RLock()
	myID := s.state.Identity.PeerID
	s.state.Mu.RUnlock()

	transferID := transport.GenerateTransferID(handler.Filename + time.Now().String())

	// Создаем папку Outbox
	outboxDir := filepath.Join(s.AccountMgr.GetAccountDir(myID), "outbox")
	os.MkdirAll(outboxDir, 0755)

	outPath := filepath.Join(outboxDir, transferID+"_"+handler.Filename)
	outFile, err := os.Create(outPath)
	if err != nil {
		http.Error(w, "Error creating local outbox file", http.StatusInternalServerError)
		return
	}

	// Безопасно переливаем гигабайты из HTTP-запроса на диск
	_, err = io.Copy(outFile, file)
	outFile.Close()
	if err != nil {
		os.Remove(outPath)
		http.Error(w, "Error writing file to disk", http.StatusInternalServerError)
		return
	}

	// Записываем в базу
	outboxFile := storage.OutboxFile{
		TransferID: transferID, TargetID: peerID,
		FilePath: outPath, FileName: handler.Filename, FileSize: handler.Size,
	}
	storage.SaveOutboxFile(outboxFile)

	// Запускаем фоновый обработчик очереди. Если пир онлайн — отправит сейчас, если нет — отправит при подключении.
	go transport.ProcessOutboxForPeer(s.state, s.Broadcast, peerID)

	json.NewEncoder(w).Encode(map[string]string{"status": "queued"})
}

// handleFileView безопасно отдает файлы из папки data активного аккаунта
func (s *Server) handleFileView(w http.ResponseWriter, r *http.Request) {
	filePath := r.URL.Query().Get("path")
	if filePath == "" {
		http.Error(w, "Path required", http.StatusBadRequest)
		return
	}

	// Очищаем путь от попыток выйти из папки (защита от ../..)
	cleanPath := filepath.Clean(filePath)

	// Проверяем, что файл находится внутри папки data
	if !strings.HasPrefix(cleanPath, "data") {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// Проверяем, не пытаются ли украсть ключи
	if strings.Contains(cleanPath, "keystore.json") {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// Добавляем заголовки для текстовых файлов (нужно для JavaScript fetch)
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Disposition", "inline")

	http.ServeFile(w, r, cleanPath)
}
