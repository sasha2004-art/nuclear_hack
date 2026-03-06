package transport

import (
	"testing"
)

func TestCalculateHash(t *testing.T) {
	content := "Hello DAG"
	parents := []string{"hash_abc", "hash_123"}

	// Тест 1: Детерминированность
	hash1 := CalculateHash(content, parents)
	hash2 := CalculateHash(content, parents)

	if hash1 != hash2 {
		t.Errorf("Хэши должны быть одинаковыми. Получено: %s и %s", hash1, hash2)
	}

	// Тест 2: Изменение контента меняет хэш
	hash3 := CalculateHash("Hello DAG!", parents)
	if hash1 == hash3 {
		t.Error("Изменение текста должно менять хэш")
	}

	// Тест 3: Изменение порядка родителей меняет хэш
	hash4 := CalculateHash(content, []string{"hash_123", "hash_abc"})
	if hash1 == hash4 {
		t.Error("Изменение порядка родителей должно менять хэш")
	}
}
