package main

import (
	"bufio"
	"embed"
	"encoding/hex"
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

	"plotix_core/accounts"
	"plotix_core/api"
	"plotix_core/core"
	"plotix_core/crypto"
	"plotix_core/discovery"
	"plotix_core/models"
	"plotix_core/storage"
	"plotix_core/transport"
	"plotix_core/utils"
)

//go:embed all:ui_dist
var uiStatic embed.FS

func main() {
	utils.SetupFirewallRules()

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

	mgr := accounts.NewManager("data")
	if err := mgr.Load(); err != nil {
		log.Fatalf("[FATAL] Account manager error: %v", err)
	}

	var ident *crypto.Identity
	if len(mgr.Accounts) == 0 {
		if _, statErr := os.Stat("keystore.json"); statErr == nil {
			log.Println("[BOOT] Migrating existing identity to data/ folder...")
			var migrErr error
			ident, migrErr = mgr.MigrateExisting(".")
			if migrErr != nil {
				log.Fatalf("[FATAL] Migration error: %v", migrErr)
			}
		} else {
			log.Println("[BOOT] Creating default account...")
			_, newIdent, createErr := mgr.CreateAccount("")
			if createErr != nil {
				log.Fatalf("[FATAL] Account creation error: %v", createErr)
			}
			ident = newIdent
			mgr.ActivePeerID = ident.PeerID
			mgr.Save()
		}
	} else {
		activeID := mgr.LoadActive()
		if activeID == "" || !mgr.HasAccount(activeID) {
			activeID = mgr.Accounts[0].PeerID
		}
		mgr.ActivePeerID = activeID

		var loadErr error
		ident, loadErr = crypto.InitIdentity(mgr.GetKeystorePath(activeID))
		if loadErr != nil {
			log.Fatalf("[FATAL] Key init error: %v", loadErr)
		}
	}

	log.Printf("[BOOT] PeerID: %s", ident.PeerID)

	storage.InitDB(mgr.GetAccountDir(mgr.ActivePeerID))

	state := core.NewNodeState(ident)

	contacts, _ := storage.GetAllContacts()
	state.Mu.Lock()
	state.PeerAliases = contacts
	state.Mu.Unlock()
	state.DisplayName = func() string {
		acc := mgr.GetAccount(mgr.ActivePeerID)
		if acc == nil || acc.Ghost {
			return ""
		}
		return acc.Name
	}

	uiContent, _ := fs.Sub(uiStatic, "ui_dist")
	server := api.NewServer(state, uiContent, mgr)

	server.SwitchAccount = func(peerID string) error {
		if !mgr.HasAccount(peerID) {
			return fmt.Errorf("account not found: %s", peerID)
		}
		state.ResetConnections()
		storage.CloseDB()

		newIdent, err := crypto.InitIdentity(mgr.GetKeystorePath(peerID))
		if err != nil {
			return err
		}

		state.Mu.Lock()
		state.Identity = newIdent
		state.Mu.Unlock()

		storage.InitDB(mgr.GetAccountDir(peerID))

		contacts, _ := storage.GetAllContacts()
		state.Mu.Lock()
		state.PeerAliases = contacts
		state.Mu.Unlock()

		mgr.ActivePeerID = peerID
		mgr.Save()

		server.Broadcast <- models.WSEvent{
			Type:    "account_switched",
			Payload: map[string]string{"peer_id": peerID},
		}

		log.Printf("[BOOT] Switched to account %s", peerID)
		return nil
	}

	go transport.StartServer(state, server.Broadcast)

	discovery.Start(state, selectedIface)

	go func() {
		for ip := range state.NewPeerChan {
			log.Printf("[BOOT] Инициирую Handshake с %s", ip)
			var displayName string
			if state.DisplayName != nil {
				displayName = state.DisplayName()
			}
			state.Mu.RLock()
			// ИСПРАВЛЕНИЕ: Добавили EphemeralPub в первоначальный пакет
			h := transport.HandshakePayload{
				PeerID:       state.Identity.PeerID,
				PublicKey:    state.Identity.PublicKey,
				Name:         displayName,
				EphemeralPub: hex.EncodeToString(state.EphemeralPub),
			}
			state.Mu.RUnlock()
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
