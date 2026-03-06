package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"plotix_core/core"
	"plotix_core/crypto"
	"plotix_core/models"
	"plotix_core/storage"
)

func TestHandleGetPeers(t *testing.T) {
	// FIX: Инициализируем временную in-memory БД для тестирования
	storage.InitDB(t.TempDir())
	defer storage.CloseDB()

	state := core.NewNodeState(&crypto.Identity{PeerID: "test"})
	state.Mu.Lock()
	state.Peers["test_peer_1"] = "192.168.1.10"
	state.Mu.Unlock()

	server := NewServer(state, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/peers", nil)
	rr := httptest.NewRecorder()

	server.handleGetPeers(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Ожидался статус %v, получен %v", http.StatusOK, status)
	}

	var response map[string]PeerEntry
	err := json.NewDecoder(rr.Body).Decode(&response)
	if err != nil {
		t.Fatalf("Ошибка парсинга JSON: %v", err)
	}

	entry, ok := response["test_peer_1"]
	if !ok || entry.IP != "192.168.1.10" {
		t.Errorf("Ожидался IP 192.168.1.10 для test_peer_1, получено: %v", entry.IP)
	}
}

func TestHandleSendMessage_OfflineQueuing(t *testing.T) {
	// FIX: Инициализируем временную БД
	storage.InitDB(t.TempDir())
	defer storage.CloseDB()

	state := core.NewNodeState(&crypto.Identity{PeerID: "test"})
	server := NewServer(state, nil, nil)

	payload := models.SendMessageReq{
		PeerID:  "offline_peer_id",
		Message: "Привет, тест для оффлайна!",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/send_message", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	server.handleSendMessage(rr, req)

	// Теперь, когда пир не найден, система не должна падать с 404 (Not Found).
	// Она должна поместить сообщение в локальную БД со статусом Delivered = false
	// и вернуть статус 200 (OK).
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Ожидался статус %v, получен %v", http.StatusOK, status)
	}
}

func TestHandleSendMessage_InvalidMethod(t *testing.T) {
	storage.InitDB(t.TempDir())
	defer storage.CloseDB()

	state := core.NewNodeState(&crypto.Identity{PeerID: "test"})
	server := NewServer(state, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/send_message", nil)
	rr := httptest.NewRecorder()

	server.handleSendMessage(rr, req)

	if status := rr.Code; status != http.StatusMethodNotAllowed {
		t.Errorf("Ожидался статус %v (Method Not Allowed), получен %v", http.StatusMethodNotAllowed, status)
	}
}
