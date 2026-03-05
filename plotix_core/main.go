package main

import (
	"bufio"
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"plotix_core/api"
	"plotix_core/core"
	"plotix_core/crypto"
	"plotix_core/discovery"
	"plotix_core/transport"
	"plotix_core/utils"
)

var uiStatic embed.FS

func main() {
	ifaceFlag := flag.String("iface", "", "Network interface name")
	flag.Parse()

	selectedIface := *ifaceFlag

	if selectedIface == "" {
		ifaces, err := utils.GetInterfacesList()
		if err != nil {
			log.Fatalf("[FATAL] Cannot get interfaces: %v", err)
		}

		if len(ifaces) == 0 {
			log.Fatal("[FATAL] No active network interfaces!")
		}

		fmt.Println("=== Plotix - Select network interface ===")
		for i, iface := range ifaces {
			fmt.Printf("[%d] %s (IP: %s)\n", i+1, iface.Name, iface.IP)
		}
		fmt.Print("Enter number: ")

		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		index, err := strconv.Atoi(input)
		if err != nil || index < 1 || index > len(ifaces) {
			log.Fatal("[FATAL] Invalid choice.")
		}
		selectedIface = ifaces[index-1].Name
	}

	log.Printf("[BOOT] Interface: %s", selectedIface)

	ident, err := crypto.InitIdentity("keystore.json")
	if err != nil {
		log.Fatalf("[FATAL] Key init error: %v", err)
	}

	log.Printf("[BOOT] PeerID: %s", ident.PeerID)

	state := core.NewNodeState(ident)

	uiContent, _ := fs.Sub(uiStatic, "ui_dist")
	server := api.NewServer(state, uiContent)

	go transport.StartServer(state, server.Broadcast)

	discovery.Start(state, selectedIface)

	go func() {
		for ip := range state.NewPeerChan {
			log.Printf("[BOOT] Инициирую Handshake с %s", ip)
			h := transport.HandshakePayload{
				PeerID:    state.Identity.PeerID,
				PublicKey: state.Identity.PublicKey,
			}
			if err := transport.SendPacket(state, server.Broadcast, "", ip, "handshake", h); err != nil {
				log.Printf("[BOOT] Ошибка Handshake с %s: %v", ip, err)
			}
		}
	}()

	go func() {
		time.Sleep(1 * time.Second)
		log.Println("[UI] Opening browser...")
		openBrowser("http://localhost:8080")
	}()

	server.Start("8080")
}

func openBrowser(url string) {
	var err error
	switch runtime.GOOS {
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = exec.Command("xdg-open", url).Start()
	}
	if err != nil {
		log.Printf("[UI] Could not open browser: %v", err)
	}
}
