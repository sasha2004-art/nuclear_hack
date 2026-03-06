package transport

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"plotix_core/core"
	"plotix_core/crypto"
	"plotix_core/models"
	"plotix_core/storage"
)

const tcpPort = "10000"

// writeMu хранит мьютексы для каждого соединения, предотвращая наложение пакетов
var writeMu sync.Map

// sendDataSafe выстраивает пакеты в очередь, гарантируя целостность потока
func sendDataSafe(conn net.Conn, data []byte) error {
	m, _ := writeMu.LoadOrStore(conn, &sync.Mutex{})
	mu := m.(*sync.Mutex)

	mu.Lock()
	defer mu.Unlock()

	// ИСПРАВЛЕНИЕ: 15 секунд вместо 5
	conn.SetWriteDeadline(time.Now().Add(15 * time.Second))
	defer conn.SetWriteDeadline(time.Time{})

	if err := binary.Write(conn, binary.BigEndian, int32(len(data))); err != nil {
		return err
	}
	if _, err := conn.Write(data); err != nil {
		return err
	}
	return nil
}

func StartServer(state *core.NodeState, uiEvents chan models.WSEvent) {
	ln, err := net.Listen("tcp", ":"+tcpPort)
	if err != nil {
		log.Fatalf("[TRANSPORT] TCP listen error: %v", err)
	}
	log.Printf("[TRANSPORT] TCP server on port %s", tcpPort)

	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}
		go handleConnection(conn, state, uiEvents)
	}
}

func handleConnection(conn net.Conn, state *core.NodeState, uiEvents chan models.WSEvent) {
	var remotePeerID string

	defer func() {
		if remotePeerID != "" {
			// Удаляем соединение, только если оно не было заменено новым
			if state.GetConnection(remotePeerID) == conn {
				state.RemoveConnection(remotePeerID)
			}
			log.Printf("[TRANSPORT] Connection with %s closed", remotePeerID)
		}
		writeMu.Delete(conn)
		conn.Close()
	}()

	for {
		var size int32
		err := binary.Read(conn, binary.BigEndian, &size)
		if err != nil {
			break
		}

		if size > 10*1024*1024 || size < 0 {
			log.Printf("[TRANSPORT] Invalid packet size: %d. Closing connection.", size)
			break
		}

		payload := make([]byte, size)
		_, err = io.ReadFull(conn, payload)
		if err != nil {
			break
		}

		var packet Packet
		if err := json.Unmarshal(payload, &packet); err != nil {
			continue
		}

		switch packet.Type {
		case "handshake":
			var h HandshakePayload
			json.Unmarshal(packet.Payload, &h)

			isInitial := (remotePeerID == "")
			remotePeerID = h.PeerID

			state.UpdatePeer(h.PeerID, conn.RemoteAddr().(*net.TCPAddr).IP.String())
			state.SetPeerName(h.PeerID, h.Name)
			state.SetPeerPubKey(h.PeerID, h.PublicKey)

			// Пытаемся сохранить соединение
			if !state.SaveConnection(h.PeerID, conn) {
				// Если SaveConnection вернул false, значит у нас уже есть активная связь с этим пиром.
				// Закрываем это (входящее) соединение, чтобы не плодить дубли.
				log.Printf("[TRANSPORT] Duplicate connection from %s rejected. Already connected.", h.PeerID)
				return
			}

			if h.EphemeralPub != "" {
				peerPub, _ := hex.DecodeString(h.EphemeralPub)
				state.Mu.RLock()
				mySec := state.EphemeralPriv
				state.Mu.RUnlock()

				sharedKey := crypto.ComputeSharedSecret(mySec, peerPub)
				if sharedKey != nil {
					state.SetSessionKey(h.PeerID, sharedKey)
				}
			}

			log.Printf("[SECURITY] SECURE E2EE Handshake OK: %s.", h.PeerID)

			if isInitial {
				state.Mu.RLock()
				var displayName string
				if state.DisplayName != nil {
					displayName = state.DisplayName()
				}
				ackH := HandshakePayload{
					PeerID:       state.Identity.PeerID,
					PublicKey:    state.Identity.PublicKey,
					Name:         displayName,
					EphemeralPub: hex.EncodeToString(state.EphemeralPub),
				}
				state.Mu.RUnlock()
				hData, _ := json.Marshal(ackH)
				ackPacket := Packet{Type: "handshake_ack", Payload: hData}
				packetData, _ := json.Marshal(ackPacket)
				sendDataSafe(conn, packetData)
			}

			go ResendPendingMessages(state, h.PeerID, uiEvents)
			go ProcessOutboxForPeer(state, uiEvents, h.PeerID)

			go func() {
				myHeads := storage.GetHeads(h.PeerID)
				state.Mu.RLock()
				myID := state.Identity.PeerID
				state.Mu.RUnlock()
				syncReq := SyncRequestPayload{PeerID: myID, Heads: myHeads}
				reqData, _ := json.Marshal(syncReq)
				syncPacket := Packet{Type: "sync_request", Payload: reqData}
				packetData, _ := json.Marshal(syncPacket)
				sendDataSafe(conn, packetData)
			}()

		case "handshake_ack":
			var h HandshakePayload
			json.Unmarshal(packet.Payload, &h)
			remotePeerID = h.PeerID

			state.UpdatePeer(h.PeerID, conn.RemoteAddr().(*net.TCPAddr).IP.String())
			state.SetPeerName(h.PeerID, h.Name)
			state.SetPeerPubKey(h.PeerID, h.PublicKey)

			if !state.SaveConnection(h.PeerID, conn) {
				log.Printf("[TRANSPORT] Duplicate connection ACK from %s rejected. Already connected.", h.PeerID)
				return
			}

			if h.EphemeralPub != "" {
				peerPub, _ := hex.DecodeString(h.EphemeralPub)
				state.Mu.RLock()
				mySec := state.EphemeralPriv
				state.Mu.RUnlock()

				sharedKey := crypto.ComputeSharedSecret(mySec, peerPub)
				if sharedKey != nil {
					state.SetSessionKey(h.PeerID, sharedKey)
				}
			}

			log.Printf("[SECURITY] SECURE E2EE Handshake ACK Received: %s. Ready.", h.PeerID)

			go ProcessOutboxForPeer(state, uiEvents, h.PeerID)

		case "ack":
			var msgID string
			json.Unmarshal(packet.Payload, &msgID)
			if remotePeerID != "" {
				storage.MarkDelivered(remotePeerID, msgID)
			}

		case "chat":
			var c ChatPayload
			json.Unmarshal(packet.Payload, &c)
			processIncomingChat(c, conn, state, uiEvents, remotePeerID)

		case "sync_request":
			var req SyncRequestPayload
			json.Unmarshal(packet.Payload, &req)

			if remotePeerID == "" {
				remotePeerID = req.PeerID
			}

			missingEntities := findMissingMessages(remotePeerID, req.Heads)
			if len(missingEntities) > 0 {
				state.Mu.RLock()
				myID := state.Identity.PeerID
				privKey := state.Identity.PrivateKey
				state.Mu.RUnlock()
				sessionKey := state.GetSessionKey(remotePeerID)

				var secureMessages []ChatPayload
				for _, msg := range missingEntities {
					chat := ChatPayload{
						ID: msg.ID, Parents: msg.Parents, Content: msg.Text,
						SenderID: msg.Sender, TargetID: remotePeerID, Timestamp: msg.Timestamp,
					}
					if sessionKey != nil {
						ctxt, nonce, _ := crypto.EncryptAES(sessionKey, []byte(chat.Content))
						chat.Content = hex.EncodeToString(ctxt)
						chat.Nonce = hex.EncodeToString(nonce)
					}
					chat.Signature = crypto.SignMessage(privKey, chat.ID)
					secureMessages = append(secureMessages, chat)
				}

				resp := SyncResponsePayload{PeerID: myID, Messages: secureMessages}
				data, _ := json.Marshal(resp)
				respPacket := Packet{Type: "sync_response", Payload: data}
				packetData, _ := json.Marshal(respPacket)
				sendDataSafe(conn, packetData)
			}

		case "sync_response":
			var resp SyncResponsePayload
			json.Unmarshal(packet.Payload, &resp)
			for _, c := range resp.Messages {
				processIncomingChat(c, conn, state, uiEvents, remotePeerID)
			}

		case "file_start":
			var fs FileStartPayload
			json.Unmarshal(packet.Payload, &fs)

			state.Mu.RLock()
			myID := state.Identity.PeerID
			state.Mu.RUnlock()

			if fs.TargetID != myID {
				continue
			}

			err := InitIncomingFile(fs.TransferID, fs.FileName, fs.FileSize, myID)
			if err != nil {
				log.Printf("[FILE] Ошибка инициализации файла: %v", err)
			}

		case "file_chunk":
			var fc FileChunkPayload
			if err := json.Unmarshal(packet.Payload, &fc); err != nil {
				log.Printf("[FILE] Ошибка парсинга чанка: %v", err)
				continue
			}

			senderPubKey := state.GetPeerPubKey(remotePeerID)
			if senderPubKey != "" && !crypto.VerifySignature(senderPubKey, fc.TransferID, fc.Signature) {
				log.Printf("[SECURITY] Подпись чанка файла недействительна!")
				continue
			}

			var chunkData []byte
			sessionKey := state.GetSessionKey(remotePeerID)

			if sessionKey != nil && len(fc.Nonce) > 0 {
				plaintext, err := crypto.DecryptAES(sessionKey, fc.Nonce, fc.Data)
				if err != nil {
					log.Printf("[SECURITY] E2EE ошибка расшифровки чанка. Сохраняем как есть. Ошибка: %v", err)
					chunkData = fc.Data // Фолбэк, чтобы файл не оборвался
				} else {
					chunkData = plaintext
				}
			} else {
				chunkData = fc.Data
			}

			done, savedPath := WriteChunk(fc.TransferID, chunkData, fc.TotalChunks)
			if done {
				msgID := CalculateHash(fc.TransferID, []string{})
				now := time.Now().UnixMilli()
				fileMsg := "[ФАЙЛ ПОЛУЧЕН] " + savedPath

				entity := storage.MessageEntity{
					ID: msgID, Parents: []string{}, Sender: remotePeerID, Text: fileMsg,
					Timestamp: now, Delivered: true,
				}
				storage.SaveMessage(remotePeerID, entity)
				state.SetLastMsgID(remotePeerID, msgID)

				if uiEvents != nil {
					uiEvents <- models.WSEvent{
						Type: "new_message",
						Payload: map[string]interface{}{
							"id": msgID, "sender": remotePeerID, "text": fileMsg, "timestamp": now,
						},
					}
				}
			}

		case "webrtc_signal":
			var sig WebRTCSignalPayload
			json.Unmarshal(packet.Payload, &sig)

			// Просто пробрасываем событие в браузер через WebSocket
			if uiEvents != nil {
				uiEvents <- models.WSEvent{
					Type:    "webrtc_signal",
					Payload: sig,
				}
			}
		}
	}
}

func processIncomingChat(c ChatPayload, conn net.Conn, state *core.NodeState, uiEvents chan models.WSEvent, remotePeerID string) {
	state.Mu.RLock()
	myID := state.Identity.PeerID
	state.Mu.RUnlock()

	if c.TargetID != "" && c.TargetID != myID {
		return
	}

	senderID := c.SenderID
	if senderID == "" {
		senderID = remotePeerID
	}

	senderPubKey := state.GetPeerPubKey(senderID)
	if senderPubKey != "" && !crypto.VerifySignature(senderPubKey, c.ID, c.Signature) {
		log.Printf("[SECURITY] ALERT! Invalid signature from %s. Message dropped.", senderID)
		return
	}

	sessionKey := state.GetSessionKey(senderID)
	if sessionKey != nil && c.Nonce != "" {
		ctxt, _ := hex.DecodeString(c.Content)
		nonce, _ := hex.DecodeString(c.Nonce)
		plaintext, err := crypto.DecryptAES(sessionKey, nonce, ctxt)
		if err != nil {
			log.Printf("[SECURITY] Decryption failed for message %s: %v", c.ID, err)
			return
		}
		c.Content = string(plaintext)
	}

	if storage.MessageExists(senderID, c.ID) {
		sendAck(conn, c.ID)
		return
	}

	remoteIP := conn.RemoteAddr().(*net.TCPAddr).IP.String()
	state.UpdatePeer(senderID, remoteIP)

	log.Printf("[CHAT] Secured Msg from %s: %s", senderID, c.Content)

	entity := storage.MessageEntity{
		ID: c.ID, Parents: c.Parents, Sender: senderID, Text: c.Content,
		Timestamp: c.Timestamp, Delivered: true,
	}
	storage.SaveMessage(senderID, entity)
	state.SetLastMsgID(senderID, c.ID)

	if uiEvents != nil {
		uiEvents <- models.WSEvent{
			Type: "new_message",
			Payload: map[string]interface{}{
				"id": c.ID, "sender": senderID, "text": c.Content, "timestamp": c.Timestamp,
			},
		}
	}
	sendAck(conn, c.ID)
}

func findMissingMessages(peerID string, remoteHeads []string) []storage.MessageEntity {
	history, _ := storage.GetHistory(peerID)
	if len(history) == 0 {
		return nil
	}
	known := make(map[string]bool)
	for _, h := range remoteHeads {
		known[h] = true
	}
	var missing []storage.MessageEntity
	for i := len(history) - 1; i >= 0; i-- {
		msg := history[i]
		if known[msg.ID] {
			break
		}
		missing = append(missing, msg)
	}
	return missing
}

func sendAck(conn net.Conn, msgID string) {
	data, _ := json.Marshal(msgID)
	packet := Packet{Type: "ack", Payload: data}
	packetData, _ := json.Marshal(packet)
	sendDataSafe(conn, packetData)
}

func ResendPendingMessages(state *core.NodeState, peerID string, uiEvents chan models.WSEvent) {
	pending, err := storage.GetPendingMessages(peerID)
	if err != nil || len(pending) == 0 {
		return
	}

	state.Mu.RLock()
	myID := state.Identity.PeerID
	ip := state.Peers[peerID]
	state.Mu.RUnlock()

	for _, msg := range pending {
		chat := ChatPayload{
			ID: msg.ID, Parents: msg.Parents, Content: msg.Text,
			SenderID: myID, TargetID: peerID, Timestamp: msg.Timestamp,
		}
		if err := SendPacket(state, uiEvents, peerID, ip, "chat", chat); err != nil {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func SendPacket(state *core.NodeState, uiEvents chan models.WSEvent, targetPeerID string, ip string, pType string, payload interface{}) error {
	conn := state.GetConnection(targetPeerID)

	if conn == nil {
		newConn, err := net.DialTimeout("tcp", ip+":"+tcpPort, 2*time.Second)
		if err != nil {
			return err
		}

		var displayName string
		if state.DisplayName != nil {
			displayName = state.DisplayName()
		}

		state.Mu.RLock()
		h := HandshakePayload{
			PeerID:       state.Identity.PeerID,
			PublicKey:    state.Identity.PublicKey,
			Name:         displayName,
			EphemeralPub: hex.EncodeToString(state.EphemeralPub),
		}
		state.Mu.RUnlock()

		hData, _ := json.Marshal(h)
		hPacket := Packet{Type: "handshake", Payload: hData}
		hBytes, _ := json.Marshal(hPacket)

		if err := sendDataSafe(newConn, hBytes); err != nil {
			newConn.Close()
			return err
		}

		if targetPeerID != "" {
			ok := state.SaveConnection(targetPeerID, newConn)
			if !ok {
				// Соединение уже существует, закрываем это новое
				log.Printf("[TRANSPORT] Connection to %s already exists, rejecting new dial.", targetPeerID)
				newConn.Close()
				// Используем старое соединение
				if oldConn := state.GetConnection(targetPeerID); oldConn != nil {
					conn = oldConn
				} else {
					return fmt.Errorf("connection to %s already exists but became unavailable", targetPeerID)
				}
			} else {
				go handleConnection(newConn, state, uiEvents)
				conn = newConn
			}
		} else {
			go handleConnection(newConn, state, uiEvents)
			conn = newConn
		}

		// Умное ожидание установки ключа: ждем ACK от другой стороны до 2 секунд
		for i := 0; i < 20; i++ {
			if state.GetSessionKey(targetPeerID) != nil {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
	}

	if pType == "chat" {
		if chat, ok := payload.(ChatPayload); ok {
			state.Mu.RLock()
			privKeyHex := state.Identity.PrivateKey
			state.Mu.RUnlock()

			chat.Signature = crypto.SignMessage(privKeyHex, chat.ID)

			sessionKey := state.GetSessionKey(targetPeerID)
			if sessionKey != nil {
				ctxt, nonce, _ := crypto.EncryptAES(sessionKey, []byte(chat.Content))
				chat.Content = hex.EncodeToString(ctxt)
				chat.Nonce = hex.EncodeToString(nonce)
			} else {
				log.Println("[SECURITY] Warning: Sending unencrypted (No session key yet)")
			}
			payload = chat
		}
	}

	data, _ := json.Marshal(payload)
	packet := Packet{
		Type:    pType,
		Payload: data,
	}
	packetData, _ := json.Marshal(packet)

	if err := sendDataSafe(conn, packetData); err != nil {
		conn.Close()
		state.RemoveConnection(targetPeerID)
		return err
	}

	return nil
}
