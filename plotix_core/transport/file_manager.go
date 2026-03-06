package transport

import (
	"crypto/sha256"
	"encoding/hex"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"plotix_core/core"
	"plotix_core/crypto"
	"plotix_core/models"
	"plotix_core/storage"
)

type IncomingFile struct {
	File       *os.File
	FilePath   string
	FileName   string
	FileSize   int64
	ChunksRcvd int
}

var (
	incomingFiles = make(map[string]*IncomingFile)
	fileMu        sync.Mutex
)

// InitIncomingFile подготавливает файл к записи на диске получателя
func InitIncomingFile(transferID, fileName string, fileSize int64, myPeerID string) error {
	fileMu.Lock()
	defer fileMu.Unlock()

	// Создаем папку загрузок для активного аккаунта
	downloadDir := filepath.Join("data", myPeerID, "downloads")
	os.MkdirAll(downloadDir, 0755)

	filePath := filepath.Join(downloadDir, fileName)

	// Если файл существует, добавляем суффикс
	if _, err := os.Stat(filePath); err == nil {
		filePath = filepath.Join(downloadDir, transferID[:8]+"_"+fileName)
	}

	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	incomingFiles[transferID] = &IncomingFile{
		File:     f,
		FilePath: filePath,
		FileName: fileName,
		FileSize: fileSize,
	}

	log.Printf("[FILE] Начата загрузка: %s (%.2f MB)", fileName, float64(fileSize)/1024/1024)
	return nil
}

// WriteChunk записывает расшифрованный блок в файл
func WriteChunk(transferID string, data []byte, totalChunks int) (bool, string) {
	fileMu.Lock()
	defer fileMu.Unlock()

	info, exists := incomingFiles[transferID]
	if !exists {
		return false, ""
	}

	info.File.Write(data)
	info.ChunksRcvd++

	// Если это последний чанк
	if info.ChunksRcvd >= totalChunks {
		info.File.Close()
		delete(incomingFiles, transferID)
		log.Printf("[FILE] Загрузка завершена: %s", info.FilePath)
		return true, info.FilePath
	}

	return false, ""
}

// GenerateTransferID создает уникальный ID для передачи
func GenerateTransferID(filePath string) string {
	h := sha256.New()
	h.Write([]byte(filePath))
	return hex.EncodeToString(h.Sum(nil))
}

var outboxMu sync.Mutex

func ProcessOutboxForPeer(state *core.NodeState, uiEvents chan models.WSEvent, peerID string) {
	outboxMu.Lock()
	defer outboxMu.Unlock()

	files := storage.GetOutboxFiles(peerID)
	if len(files) == 0 {
		return
	}

	state.Mu.RLock()
	myID := state.Identity.PeerID
	privKey := state.Identity.PrivateKey
	ip, online := state.Peers[peerID]
	state.Mu.RUnlock()

	if !online || ip == "" {
		return
	}

	for _, f := range files {
		startPayload := FileStartPayload{
			TransferID: f.TransferID, FileName: f.FileName,
			FileSize: f.FileSize, SenderID: myID, TargetID: f.TargetID,
		}

		err := SendPacket(state, uiEvents, peerID, ip, "file_start", startPayload)
		if err != nil {
			log.Printf("[FILE] Сеть недоступна, файл %s остается в очереди", f.FileName)
			return
		}

		fileObj, err := os.Open(f.FilePath)
		if err != nil {
			log.Printf("[FILE] Не удалось прочитать локальный файл: %v", err)
			continue
		}

		chunkSize := 256 * 1024
		buffer := make([]byte, chunkSize)

		// Защита от деления на 0 при пустых файлах
		totalChunks := 1
		if f.FileSize > 0 {
			totalChunks = int((f.FileSize + int64(chunkSize) - 1) / int64(chunkSize))
		}

		transferSuccess := true

		for i := 0; i < totalChunks; i++ {
			n, err := fileObj.Read(buffer)
			if err != nil && err.Error() != "EOF" {
				log.Printf("[FILE] Ошибка чтения куска файла: %v", err)
				transferSuccess = false
				break
			}

			if n == 0 && f.FileSize > 0 {
				break
			}

			chunkPayload := FileChunkPayload{
				TransferID: f.TransferID, ChunkIndex: i, TotalChunks: totalChunks,
			}

			currentSessionKey := state.GetSessionKey(peerID)

			if currentSessionKey != nil && len(currentSessionKey) == 32 {
				ctxt, nonce, encErr := crypto.EncryptAES(currentSessionKey, buffer[:n])
				if encErr != nil {
					log.Printf("[SECURITY] Ошибка шифрования чанка: %v", encErr)
					chunkPayload.Data = buffer[:n]
				} else {
					chunkPayload.Data = ctxt
					chunkPayload.Nonce = nonce
				}
			} else {
				chunkPayload.Data = buffer[:n]
			}

			chunkPayload.Signature = crypto.SignMessage(privKey, f.TransferID)

			err = SendPacket(state, uiEvents, peerID, ip, "file_chunk", chunkPayload)
			if err != nil {
				log.Printf("[FILE] Ошибка отправки чанка %d: %v", i, err)
				transferSuccess = false
				break
			}

			time.Sleep(10 * time.Millisecond) // Увеличил паузу до 10мс для стабильности TCP
		}
		fileObj.Close()

		if transferSuccess {
			// ИСПРАВЛЕНИЕ: Вместо удаления перемещаем в папку 'sent' для истории
			sentDir := filepath.Join(filepath.Dir(filepath.Dir(f.FilePath)), "sent")
			os.MkdirAll(sentDir, 0755)
			finalPath := filepath.Join(sentDir, filepath.Base(f.FilePath))

			os.Rename(f.FilePath, finalPath)
			storage.RemoveOutboxFile(f.TransferID)

			msgID := CalculateHash(f.TransferID, []string{})
			now := time.Now().UnixMilli()

			// ВАЖНО: Сохраняем полный путь к файлу в сообщении отправителя
			fileMsg := "[ФАЙЛ ОТПРАВЛЕН] " + finalPath

			entity := storage.MessageEntity{
				ID: msgID, Parents: []string{}, Sender: myID, Text: fileMsg,
				Timestamp: now, Delivered: true,
			}
			storage.SaveMessage(peerID, entity)
			state.SetLastMsgID(peerID, msgID)

			if uiEvents != nil {
				uiEvents <- models.WSEvent{
					Type: "new_message",
					Payload: map[string]interface{}{
						"id": msgID, "sender": myID, "text": fileMsg, "timestamp": now,
					},
				}
			}
		} else {
			return // Обрыв - выходим из функции обработки очереди
		}
	}
}
