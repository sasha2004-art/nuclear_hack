package discovery

import (
	"encoding/json"
	"log"
	"net"
	"time"

	"plotix_core/core"
	"plotix_core/utils"
)

const (
	multicastGroup = "224.0.0.251:9999"
	broadcastAddr  = "255.255.255.255:9999"
)

type AnnounceMsg struct {
	PeerID string `json:"peer_id"`
	Name   string `json:"name,omitempty"`
}

func Start(state *core.NodeState, ifaceName string) {
	iface, localIP, err := utils.GetInterfaceByName(ifaceName)
	if err != nil {
		log.Fatalf("[DISCOVERY] Ошибка интерфейса: %v", err)
	}

	mAddr, _ := net.ResolveUDPAddr("udp4", multicastGroup)
	bAddr, _ := net.ResolveUDPAddr("udp4", broadcastAddr)

	log.Printf("[DISCOVERY] Старт на %s (IP: %s)", iface.Name, localIP)

	go listen(state, iface)

	go broadcast(state, localIP, mAddr, bAddr)
}

func listen(state *core.NodeState, iface *net.Interface) {

	addr := &net.UDPAddr{IP: net.IPv4zero, Port: 9999}
	conn, err := net.ListenUDP("udp4", addr)
	if err != nil {
		log.Fatalf("[DISCOVERY] Ошибка слушателя: %v", err)
	}
	defer conn.Close()

	buffer := make([]byte, 1024)
	for {
		n, src, err := conn.ReadFromUDP(buffer)
		if err != nil {
			continue
		}

		var msg AnnounceMsg
		if err := json.Unmarshal(buffer[:n], &msg); err != nil {
			continue
		}

		state.Mu.RLock()
		selfID := state.Identity.PeerID
		state.Mu.RUnlock()

		if msg.PeerID == selfID {
			continue
		}

		state.UpdatePeer(msg.PeerID, src.IP.String())
		state.SetPeerName(msg.PeerID, msg.Name)
		log.Printf("[DISCOVERY] Получен сигнал от %s (IP: %s)", msg.PeerID, src.IP.String())
	}
}

func broadcast(state *core.NodeState, localIP net.IP, mAddr, bAddr *net.UDPAddr) {
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: localIP, Port: 0})
	if err != nil {
		log.Fatalf("[DISCOVERY] Ошибка вещателя: %v", err)
	}
	defer conn.Close()

	for {
		state.Mu.RLock()
		peerID := state.Identity.PeerID
		state.Mu.RUnlock()

		var name string
		if state.DisplayName != nil {
			name = state.DisplayName()
		}

		msg := AnnounceMsg{PeerID: peerID, Name: name}
		data, _ := json.Marshal(msg)

		conn.WriteToUDP(data, mAddr)
		conn.WriteToUDP(data, bAddr)

		time.Sleep(3 * time.Second)
	}
}
