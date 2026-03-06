package transport

import (
	"os"
	"testing"
)

func TestFileChunkingAndAssembly(t *testing.T) {
	myPeerID := "test_file_peer"
	transferID := "test_transfer_123"
	fileName := "test_image.jpg"

	// Очистка перед тестом
	os.RemoveAll("data")

	err := InitIncomingFile(transferID, fileName, 20, myPeerID)
	if err != nil {
		t.Fatalf("Ошибка инициализации файла: %v", err)
	}

	// Имитация получения 2 чанков
	chunk1 := []byte("Hello ")
	chunk2 := []byte("World!")
	totalChunks := 2

	done1, _ := WriteChunk(transferID, chunk1, totalChunks)
	if done1 {
		t.Error("Файл пометился как завершенный слишком рано")
	}

	done2, savedPath := WriteChunk(transferID, chunk2, totalChunks)
	if !done2 {
		t.Error("Файл должен был завершиться")
	}

	if savedPath == "" {
		t.Error("Путь сохранения пуст")
	}

	// Читаем собранный файл
	assembled, err := os.ReadFile(savedPath)
	if err != nil {
		t.Fatalf("Не удалось прочитать собранный файл: %v", err)
	}

	if string(assembled) != "Hello World!" {
		t.Errorf("Ожидалось 'Hello World!', получено: '%s'", string(assembled))
	}

	// Очистка после теста
	os.RemoveAll("data")
}
