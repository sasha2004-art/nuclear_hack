package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"plotix_core/core"
	"plotix_core/models"
)

func TestHandleGetPeers(t *testing.T) {
	state := core.NewNodeState()
	state.Mu.Lock()
	state.Peers["test_peer_1"] = "192.168.1.10"
	state.Mu.Unlock()

	server := NewServer(state)

	req := httptest.NewRequest(http.MethodGet, "/peers", nil)
	rr := httptest.NewRecorder()

	server.handleGetPeers(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Ожидался статус %v, получен %v", http.StatusOK, status)
	}

	var response map[string]string
	err := json.NewDecoder(rr.Body).Decode(&response)
	if err != nil {
		t.Fatalf("Ошибка парсинга JSON: %v", err)
	}

	if ip, ok := response["test_peer_1"]; !ok || ip != "192.168.1.10" {
		t.Errorf("Ожидался IP 192.168.1.10 для test_peer_1, получено: %v", ip)
	}
}

func TestHandleSendMessage_Valid(t *testing.T) {
	state := core.NewNodeState()
	server := NewServer(state)

	payload := models.SendMessageReq{
		PeerID:  "target_user",
		Message: "Привет, тест!",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/send_message", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	server.handleSendMessage(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Ожидался статус %v, получен %v", http.StatusOK, status)
	}

	var response map[string]string
	json.NewDecoder(rr.Body).Decode(&response)
	if response["status"] != "ok" {
		t.Errorf("Ожидался статус 'ok', получено: %v", response["status"])
	}
}

func TestHandleSendMessage_InvalidMethod(t *testing.T) {
	state := core.NewNodeState()
	server := NewServer(state)

	req := httptest.NewRequest(http.MethodGet, "/send_message", nil)
	rr := httptest.NewRecorder()

	server.handleSendMessage(rr, req)

	if status := rr.Code; status != http.StatusMethodNotAllowed {
		t.Errorf("Ожидался статус %v (Method Not Allowed), получен %v", http.StatusMethodNotAllowed, status)
	}
}
