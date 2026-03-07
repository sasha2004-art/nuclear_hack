package discovery

import (
	"encoding/json"
	"log"
	"net" // Используем стандартный net для типов и UDP
	"time"

	"github.com/wlynxg/anet" // Исправляет Permission Denied на Android
	"plotix_core/core"
)

const (
	multicastGroup = "224.0.0.251:9999"
	broadcastAddr  = "255.255.255.255:9999"
)

// AnnounceMsg структура сообщения для обнаружения узлов
type AnnounceMsg struct {
	PeerID string `json:"peer_id"`
	Name   string `json:"name,omitempty"`
}

// Start запускает процесс обнаружения
func Start(state *core.NodeState, ifaceName string) {
	// 1. Используем anet для безопасного получения интерфейса на Android
	iface, err := anet.InterfaceByName(ifaceName)
	if err != nil {
		log.Printf("[DISCOVERY] Ошибка поиска интерфейса %s: %v", ifaceName, err)
		return
	}

	// 2. Используем anet для получения IP-адресов этого интерфейса
	// Это заменяет net.InterfaceAddrs(), который запрещен на Android 11+
	addrs, err := anet.InterfaceAddrsByInterface(iface)
	if err != nil || len(addrs) == 0 {
		log.Printf("[DISCOVERY] Не удалось получить IP для %s: %v", ifaceName, err)
		return
	}

	var localIP net.IP
	for _, addr := range addrs {
		// Ищем первый подходящий IPv4 адрес
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				localIP = ipnet.IP
				break
			}
		}
	}

	if localIP == nil {
		log.Printf("[DISCOVERY] IPv4 адрес не найден на %s", ifaceName)
		return
	}

	mAddr, _ := net.ResolveUDPAddr("udp4", multicastGroup)
	bAddr, _ := net.ResolveUDPAddr("udp4", broadcastAddr)

	log.Printf("[DISCOVERY] Старт на %s (IP: %s)", iface.Name, localIP)

	// Запускаем слушатель (принимает чужие сигналы)
	go listen(state, iface)

	// Запускаем вещатель (отправляет наши сигналы)
	go broadcast(state, localIP, mAddr, bAddr)
}

// listen слушает входящие UDP пакеты
func listen(state *core.NodeState, iface *net.Interface) {
	// Слушаем на всех интерфейсах (0.0.0.0), порт 9999
	addr := &net.UDPAddr{IP: net.IPv4zero, Port: 9999}

	// Обычный ListenUDP разрешен на Android
	conn, err := net.ListenUDP("udp4", addr)
	if err != nil {
		log.Printf("[DISCOVERY] Ошибка слушателя: %v", err)
		return
	}
	defer conn.Close()

	buffer := make([]byte, 1024)
	for {
		n, src, err := conn.ReadFromUDP(buffer)
		if err != nil {
			log.Printf("[DISCOVERY] Ошибка чтения: %v", err)
			continue
		}

		var msg AnnounceMsg
		if err := json.Unmarshal(buffer[:n], &msg); err != nil {
			continue
		}

		state.Mu.RLock()
		selfID := state.Identity.PeerID
		state.Mu.RUnlock()

		// Игнорируем собственные пакеты
		if msg.PeerID == selfID {
			continue
		}

		// Обновляем данные об узле в состоянии
		state.UpdatePeer(msg.PeerID, src.IP.String())
		if msg.Name != "" {
			state.SetPeerName(msg.PeerID, msg.Name)
		}
		state.UpdateLastSeen(msg.PeerID)

		log.Printf("[DISCOVERY] Найден пир: %s (IP: %s)", msg.PeerID, src.IP.String())
	}
}

// broadcast отправляет пакеты анонса в сеть
func broadcast(state *core.NodeState, localIP net.IP, mAddr, bAddr *net.UDPAddr) {
	// Привязываемся к локальному IP для отправки
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: localIP, Port: 0})
	if err != nil {
		log.Printf("[DISCOVERY] Ошибка вещателя: %v", err)
		return
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

		msg := AnnounceMsg{
			PeerID: peerID,
			Name:   name,
		}
		data, _ := json.Marshal(msg)

		// Отправляем и в мультикаст, и в броадкаст для надежности
		conn.WriteToUDP(data, mAddr)
		conn.WriteToUDP(data, bAddr)

		time.Sleep(5 * time.Second)
	}
}