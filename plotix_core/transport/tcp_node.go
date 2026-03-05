package transport

import (
	"encoding/binary"
	"encoding/json"
	"io"
	"log"
	"net"

	"plotix_core/core"
	"plotix_core/models"
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
	var currentPeerID string

	defer func() {
		if currentPeerID != "" {
			state.RemoveConnection(currentPeerID)
			log.Printf("[TRANSPORT] Connection with %s closed", currentPeerID)
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
			currentPeerID = h.PeerID

			state.UpdatePeer(h.PeerID, conn.RemoteAddr().(*net.TCPAddr).IP.String())
			state.SaveConnection(h.PeerID, conn)

			log.Printf("[TRANSPORT] Handshake OK, persistent channel with %s saved", h.PeerID)

		case "chat":
			var c ChatPayload
			json.Unmarshal(packet.Payload, &c)

			senderID := currentPeerID
			if senderID == "" {
				remoteIP := conn.RemoteAddr().(*net.TCPAddr).IP.String()
				state.Mu.RLock()
				for id, ip := range state.Peers {
					if ip == remoteIP {
						senderID = id
						break
					}
				}
				state.Mu.RUnlock()
			}

			log.Printf("[CHAT] From %s: %s", senderID, c.Content)

			if senderID != "" && uiEvents != nil {
				uiEvents <- models.WSEvent{
					Type: "new_message",
					Payload: map[string]string{
						"sender": senderID,
						"text":   c.Content,
					},
				}
			}
		}
	}
}

func SendPacket(state *core.NodeState, uiEvents chan models.WSEvent, peerID string, ip string, pType string, payload interface{}) error {
	data, _ := json.Marshal(payload)
	packet := Packet{
		Type:    pType,
		Payload: data,
	}
	packetData, _ := json.Marshal(packet)

	conn := state.GetConnection(peerID)

	if conn == nil {
		var err error
		conn, err = net.Dial("tcp", ip+":"+tcpPort)
		if err != nil {
			return err
		}
		if peerID != "" {
			state.SaveConnection(peerID, conn)
		}
		go handleConnection(conn, state, uiEvents)
	}

	err := binary.Write(conn, binary.BigEndian, int32(len(packetData)))
	if err != nil {
		if peerID != "" {
			state.RemoveConnection(peerID)
		}
		return err
	}

	_, err = conn.Write(packetData)
	if err != nil {
		if peerID != "" {
			state.RemoveConnection(peerID)
		}
	}

	return err
}
