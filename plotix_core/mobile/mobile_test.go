package mobile

import (
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"plotix_core/accounts"
	"plotix_core/crypto"
)

// TestStartNode_FullCycle проверяет первичную инициализацию ядра в чистой папке.
// Запускает реальные серверы, проверяет создание файлов и открытие портов.
func TestStartNode_FullCycle(t *testing.T) {
	// 1. Создаем временную директорию (имитация песочницы мобильного приложения)
	tmpDir, err := os.MkdirTemp("", "plotix_mobile_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	t.Logf("[TEST] Using data dir: %s", tmpDir)

	// 2. Запускаем ноду
	status := StartNode(tmpDir)
	if status != "started" {
		t.Fatalf("Expected status 'started', got '%s'", status)
	}

	// Даем серверам время на запуск
	time.Sleep(1 * time.Second)

	// 3. ПРОВЕРКА ФАЙЛОВОЙ СИСТЕМЫ
	accFile := filepath.Join(tmpDir, "accounts.json")
	if _, err := os.Stat(accFile); os.IsNotExist(err) {
		t.Error("accounts.json was not created in data dir")
	}

	activePath := filepath.Join(tmpDir, "active.txt")
	peerID, err := os.ReadFile(activePath)
	if err != nil {
		t.Error("active.txt was not created")
	} else if len(peerID) == 0 {
		t.Error("active.txt is empty")
	} else {
		t.Logf("[OK] PeerID saved: %s", string(peerID))
	}

	// 4. ПРОВЕРКА СЕТЕВЫХ ПОРТОВ
	checkPort(t, "127.0.0.1:8080", "API Server")
	checkPort(t, "127.0.0.1:10000", "P2P Transport")

	// 5. ПРОВЕРКА ГЕТТЕРОВ
	gotPeerID := GetPeerID()
	if gotPeerID == "" {
		t.Error("GetPeerID() returned empty string")
	} else if gotPeerID != string(peerID) {
		t.Errorf("GetPeerID() = %s, want %s", gotPeerID, string(peerID))
	}

	if port := GetAPIPort(); port != "8080" {
		t.Errorf("GetAPIPort() = %s, want 8080", port)
	}
}

// TestPersistence проверяет сохранение аккаунта между "перезагрузками" приложения.
// Не поднимает серверы, работает только с файловой системой через accounts.Manager.
func TestPersistence(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "plotix_persistence_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Первый запуск: создаем аккаунт
	mgr1 := accounts.NewManager(tmpDir)
	mgr1.Load()

	_, ident, err := mgr1.CreateAccount("Test User")
	if err != nil {
		t.Fatalf("CreateAccount: %v", err)
	}
	mgr1.ActivePeerID = ident.PeerID
	mgr1.Save()

	t.Logf("[OK] First init PeerID: %s", ident.PeerID)

	// Имитируем перезагрузку: новый Manager, тот же путь
	mgr2 := accounts.NewManager(tmpDir)
	mgr2.Load()

	if len(mgr2.Accounts) != 1 {
		t.Fatalf("Expected 1 account after reboot, got %d", len(mgr2.Accounts))
	}

	activeID := mgr2.LoadActive()
	if activeID != ident.PeerID {
		t.Errorf("ActivePeerID changed! Original: %s, After reboot: %s", ident.PeerID, activeID)
	} else {
		t.Logf("[OK] Identity persisted: %s", activeID)
	}

	// Проверяем, что keystore загружается
	keystorePath := mgr2.GetKeystorePath(activeID)
	reloadedIdent, err := crypto.InitIdentity(keystorePath)
	if err != nil {
		t.Fatalf("Failed to reload identity: %v", err)
	}
	if reloadedIdent.PeerID != ident.PeerID {
		t.Errorf("Reloaded PeerID mismatch: %s vs %s", reloadedIdent.PeerID, ident.PeerID)
	} else {
		t.Logf("[OK] Keystore reloaded successfully")
	}
}

// checkPort проверяет доступность TCP порта
func checkPort(t *testing.T, addr string, name string) {
	t.Helper()
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		t.Errorf("[FAIL] %s (%s) is not reachable: %v", name, addr, err)
	} else {
		t.Logf("[OK] %s is active on %s", name, addr)
		conn.Close()
	}
}
