package storage

import (
	"os"
	"testing"
)

func TestDBFullCycle(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "plotix_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	InitDB(tempDir)
	if db == nil {
		t.Fatal("БД не была инициализирована")
	}
	defer db.Close()

	testPeer := "hex_test_peer_456"
	msg := MessageEntity{
		ID:        "msg_hash_1",
		Parents:   []string{"root"},
		Sender:    testPeer,
		Text:      "Test message",
		Timestamp: 1672522500000,
	}

	err = SaveMessage(testPeer, msg)
	if err != nil {
		t.Fatalf("Ошибка сохранения: %v", err)
	}

	history, err := GetHistory(testPeer)
	if err != nil {
		t.Fatalf("Ошибка получения истории: %v", err)
	}

	if len(history) != 1 {
		t.Fatalf("Ожидалось 1 сообщение, получено %d", len(history))
	}

	if history[0].Text != msg.Text {
		t.Errorf("Текст не совпадает: %s != %s", msg.Text, history[0].Text)
	}

	otherHistory, _ := GetHistory("other_peer")
	if len(otherHistory) != 0 {
		t.Error("История другого пира должна быть пустой")
	}
}
