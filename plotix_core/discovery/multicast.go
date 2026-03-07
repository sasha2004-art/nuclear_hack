package discovery

import (
	"encoding/json"
	"log"
	"net"
	"time"

	"github.com/wlynxg/anet"
	"golang.org/x/net/ipv4"
	"plotix_core/core"
	"plotix_core/models"
)

const (
	multicastAddr = "239.0.0.250"
	discoveryPort = 9999
)

type AnnounceMsg struct {
	PeerID string `json:"peer_id"`
	Name   string `json:"name,omitempty"`
}

func Start(state *core.NodeState, ifaceName string, broadcastChan chan models.WSEvent) {
	state.Mu.RLock()
	selfID := state.Identity.PeerID
	state.Mu.RUnlock()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[DISCOVERY] Panic in Start: %v", r)
			}
		}()

		log.Printf("[DISCOVERY] Initializing on interface: %s", ifaceName)

		iface, err := anet.InterfaceByName(ifaceName)
		if err != nil {
			log.Printf("[DISCOVERY] Error finding interface %s: %v", ifaceName, err)
			return
		}

		addrs, err := anet.InterfaceAddrsByInterface(iface)
		if err != nil || len(addrs) == 0 {
			log.Printf("[DISCOVERY] No IP found on interface %s", ifaceName)
			return
		}

		var localIP net.IP
		var bIP net.IP
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && ipnet.IP.To4() != nil {
				localIP = ipnet.IP.To4()
				mask := ipnet.Mask
				if len(mask) == 16 {
					mask = mask[12:]
				}
				calcB := make(net.IP, 4)
				for i := 0; i < 4; i++ {
					calcB[i] = localIP[i] | ^mask[i]
				}
				bIP = calcB
				break
			}
		}

		if localIP == nil {
			log.Printf("[DISCOVERY] Could not determine local IPv4")
			return
		}

		log.Printf("[DISCOVERY] CONTEXT: MyID=%s IP=%s Bcast=%s Iface=%s", selfID[:8], localIP, bIP, iface.Name)

		mAddr := &net.UDPAddr{IP: net.ParseIP(multicastAddr), Port: discoveryPort}
		bAddr := &net.UDPAddr{IP: bIP, Port: discoveryPort}

		go listen(state, iface, selfID, broadcastChan)
		go broadcast(state, localIP, iface, mAddr, bAddr)
	}()
}

func listen(state *core.NodeState, iface *net.Interface, selfID string, broadcastChan chan models.WSEvent) {
	log.Printf("[DISCOVERY] Listening on 0.0.0.0:%d", discoveryPort)
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4zero, Port: discoveryPort})
	if err != nil {
		log.Printf("[DISCOVERY] Listen error: %v", err)
		return
	}
	defer conn.Close()

	pc := ipv4.NewPacketConn(conn)
	// Включаем Loopback: если мы видим свои же пакеты, значит сетевой стек в порядке
	if err := pc.SetMulticastLoopback(true); err != nil {
		log.Printf("[DISCOVERY] SetMulticastLoopback error: %v", err)
	}

	group := net.ParseIP(multicastAddr)
	if err := pc.JoinGroup(iface, &net.UDPAddr{IP: group}); err != nil {
		log.Printf("[DISCOVERY] JoinGroup error: %v", err)
	} else {
		log.Printf("[DISCOVERY] Joined multicast group %s", multicastAddr)
	}

	buffer := make([]byte, 2048)
	for {
		n, src, err := conn.ReadFromUDP(buffer)
		if err != nil {
			continue
		}

		var msg AnnounceMsg
		if err := json.Unmarshal(buffer[:n], &msg); err != nil {
			continue
		}

		if msg.PeerID == selfID {
			// Важный лог для отладки: если он есть, то отправка и прием внутри устройства работают
			log.Printf("[DISCOVERY] [LOOPBACK] Received own packet from %s", src.IP)
			continue
		}

		log.Printf("[DISCOVERY] >>> PEER DETECTED <<< ID=%s IP=%s Name=%s", msg.PeerID[:8], src.IP, msg.Name)

		isNew := state.UpdatePeer(msg.PeerID, src.IP.String())
		state.UpdateLastSeen(msg.PeerID)
		if msg.Name != "" {
			state.SetPeerName(msg.PeerID, msg.Name)
		}

		if isNew && broadcastChan != nil {
			select {
			case broadcastChan <- models.WSEvent{
				Type:    "peer_online",
				Payload: msg.PeerID,
			}:
			default:
				log.Println("[DISCOVERY] Broadcast channel full, skipping peer_online notification")
			}
		}
	}
}

func broadcast(state *core.NodeState, localIP net.IP, iface *net.Interface, mAddr, bAddr *net.UDPAddr) {
	log.Printf("[DISCOVERY] Starting broadcast loop (3s interval)")
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
	if err != nil {
		log.Printf("[DISCOVERY] Broadcast socket error: %v", err)
		return
	}
	defer conn.Close()

	pc := ipv4.NewPacketConn(conn)
	if err := pc.SetMulticastInterface(iface); err != nil {
		log.Printf("[DISCOVERY] SetMulticastInterface error: %v", err)
	}

	for {
		state.Mu.RLock()
		peerID := state.Identity.PeerID
		var name string
		if state.DisplayName != nil {
			name = state.DisplayName()
		}
		state.Mu.RUnlock()

		msg, _ := json.Marshal(AnnounceMsg{PeerID: peerID, Name: name})

		// Отправляем по трем каналам
		_, _ = conn.WriteToUDP(msg, mAddr)
		_, _ = conn.WriteToUDP(msg, bAddr)
		_, _ = conn.WriteToUDP(msg, &net.UDPAddr{IP: net.IPv4bcast, Port: discoveryPort})

		time.Sleep(3 * time.Second)
	}
}
