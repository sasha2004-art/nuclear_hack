package transport

import (
	"encoding/binary"
	"encoding/json"
	"io"
	"log"
	"net"

	"plotix_core/core"
)

const tcpPort = "10000"

func StartServer(state *core.NodeState) {
	ln, err := net.Listen("tcp", ":"+tcpPort)
	if err != nil {
		log.Fatalf("[TRANSPORT] Ошибка запуска TCP: %v", err)
	}
	log.Printf("[TRANSPORT] TCP сервер запущен на порту %s", tcpPort)

	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}
		go handleConnection(conn, state)
	}
}

func handleConnection(conn net.Conn, state *core.NodeState) {
	defer conn.Close()

	for {

		var size int32
		err := binary.Read(conn, binary.BigEndian, &size)
		if err != nil {
			if err != io.EOF {
				log.Printf("[TRANSPORT] Ошибка чтения размера: %v", err)
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
			log.Printf("[TRANSPORT] Получено рукопожатие от %s", h.PeerID)
			state.UpdatePeer(h.PeerID, conn.RemoteAddr().(*net.TCPAddr).IP.String())

		case "chat":
			var c ChatPayload
			json.Unmarshal(packet.Payload, &c)
			log.Printf("[CHAT] Сообщение: %s", c.Content)
		}
	}
}

func SendPacket(ip string, pType string, payload interface{}) error {
	data, _ := json.Marshal(payload)
	packet := Packet{
		Type:    pType,
		Payload: data,
	}
	packetData, _ := json.Marshal(packet)

	conn, err := net.Dial("tcp", ip+":"+tcpPort)
	if err != nil {
		return err
	}
	defer conn.Close()

	binary.Write(conn, binary.BigEndian, int32(len(packetData)))
	_, err = conn.Write(packetData)
	return err
}
