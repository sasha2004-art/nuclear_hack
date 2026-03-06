package transport

import (
	"encoding/binary"
	"encoding/json"
	"io"
	"log"
	"net"
	"time"

	"plotix_core/core"
	"plotix_core/models"
	"plotix_core/storage"
)

const tcpPort = "10000"

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
			state.RemoveConnection(remotePeerID)
			log.Printf("[TRANSPORT] Connection with %s closed", remotePeerID)
		}
		conn.Close()
	}()

	for {
		var size int32
		err := binary.Read(conn, binary.BigEndian, &size)
		if err != nil {
			if err != io.EOF {
				log.Printf("[TRANSPORT] Read size error: %v", err)
			}
			break
		}

		if size > 10*1024*1024 || size < 0 {
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
			remotePeerID = h.PeerID

			state.UpdatePeer(h.PeerID, conn.RemoteAddr().(*net.TCPAddr).IP.String())
			state.SetPeerName(h.PeerID, h.Name)
			state.SaveConnection(h.PeerID, conn)

			log.Printf("[TRANSPORT] Handshake OK: %s connected", h.PeerID)

			go ResendPendingMessages(state, h.PeerID, uiEvents)

		case "ack":
			var msgID string
			json.Unmarshal(packet.Payload, &msgID)
			if remotePeerID != "" {
				storage.MarkDelivered(remotePeerID, msgID)
				log.Printf("[DELIVERY] Confirmed message %s for %s", msgID, remotePeerID)
			}

		case "chat":
			var c ChatPayload
			json.Unmarshal(packet.Payload, &c)

			state.Mu.RLock()
			myID := state.Identity.PeerID
			state.Mu.RUnlock()

			if c.TargetID != "" && c.TargetID != myID {
				log.Printf("[SECURITY] Rejected message from %s: intended for %s, but I am %s",
					c.SenderID, c.TargetID, myID)
				continue
			}

			senderID := c.SenderID
			if senderID == "" {
				senderID = remotePeerID
			}

			if senderID != "" {
				remoteIP := conn.RemoteAddr().(*net.TCPAddr).IP.String()
				state.UpdatePeer(senderID, remoteIP)

				msgTime := c.Timestamp
				if msgTime == 0 {
					msgTime = time.Now().UnixMilli()
				}

				log.Printf("[CHAT] From %s: %s", senderID, c.Content)

				entity := storage.MessageEntity{
					ID:        c.ID,
					Parents:   c.Parents,
					Sender:    senderID,
					Text:      c.Content,
					Timestamp: msgTime,
					Delivered: true,
				}
				storage.SaveMessage(senderID, entity)
				state.SetLastMsgID(senderID, c.ID)

				if uiEvents != nil {
					uiEvents <- models.WSEvent{
						Type: "new_message",
						Payload: map[string]string{
							"sender": senderID,
							"text":   c.Content,
						},
					}
				}

				sendAck(conn, c.ID)
			}
		}
	}
}

func sendAck(conn net.Conn, msgID string) {
	data, _ := json.Marshal(msgID)
	packet := Packet{Type: "ack", Payload: data}
	packetData, _ := json.Marshal(packet)

	binary.Write(conn, binary.BigEndian, int32(len(packetData)))
	conn.Write(packetData)
}

func ResendPendingMessages(state *core.NodeState, peerID string, uiEvents chan models.WSEvent) {
	pending, err := storage.GetPendingMessages(peerID)
	if err != nil || len(pending) == 0 {
		return
	}

	log.Printf("[RETRY] Resending %d messages to %s...", len(pending), peerID)

	state.Mu.RLock()
	myID := state.Identity.PeerID
	ip := state.Peers[peerID]
	state.Mu.RUnlock()

	for _, msg := range pending {
		chat := ChatPayload{
			ID:        msg.ID,
			Parents:   msg.Parents,
			Content:   msg.Text,
			SenderID:  myID,
			TargetID:  peerID,
			Timestamp: msg.Timestamp,
		}

		if err := SendPacket(state, uiEvents, peerID, ip, "chat", chat); err != nil {
			log.Printf("[RETRY] Failed to resend %s: %v", msg.ID, err)
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
			PeerID:    state.Identity.PeerID,
			PublicKey: state.Identity.PublicKey,
			Name:      displayName,
		}
		state.Mu.RUnlock()

		hData, _ := json.Marshal(h)
		hPacket := Packet{Type: "handshake", Payload: hData}
		hBytes, _ := json.Marshal(hPacket)

		binary.Write(newConn, binary.BigEndian, int32(len(hBytes)))
		newConn.Write(hBytes)

		if targetPeerID != "" {
			state.SaveConnection(targetPeerID, newConn)
		}

		go handleConnection(newConn, state, uiEvents)
		conn = newConn
	}

	data, _ := json.Marshal(payload)
	packet := Packet{
		Type:    pType,
		Payload: data,
	}
	packetData, _ := json.Marshal(packet)

	conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	err := binary.Write(conn, binary.BigEndian, int32(len(packetData)))
	if err != nil {
		conn.Close()
		state.RemoveConnection(targetPeerID)
		return err
	}

	_, err = conn.Write(packetData)
	if err != nil {
		conn.Close()
		state.RemoveConnection(targetPeerID)
		return err
	}
	conn.SetWriteDeadline(time.Time{})

	return nil
}
