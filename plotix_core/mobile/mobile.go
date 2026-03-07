package plotix

import (
	"embed"
	"encoding/hex"
	"io/fs"
	"log"
	"os"

	_ "golang.org/x/mobile/bind"

	"plotix_core/accounts"
	"plotix_core/api"
	"plotix_core/core"
	"plotix_core/crypto"
	"plotix_core/discovery"
	"plotix_core/storage"
	"plotix_core/transport"
)

//go:embed all:ui_dist
var uiStatic embed.FS

func Start(dataDir string, ifaceName string) {
	log.Printf("[MOBILE] Core starting. Iface: %s", ifaceName)

	if err := os.Chdir(dataDir); err != nil {
		log.Printf("[MOBILE] os.Chdir error: %v", err)
	}

	// 1. Account Management
	mgr := accounts.NewManager(dataDir + "/data")
	if err := mgr.Load(); err != nil {
		log.Printf("[FATAL] Account manager error: %v", err)
		return
	}

	var ident *crypto.Identity
	if len(mgr.Accounts) == 0 {
		_, newIdent, _ := mgr.CreateAccount("")
		ident = newIdent
		mgr.ActivePeerID = ident.PeerID
		mgr.Save()
	} else {
		activeID := mgr.LoadActive()
		if activeID == "" || !mgr.HasAccount(activeID) {
			activeID = mgr.Accounts[0].PeerID
		}
		mgr.ActivePeerID = activeID
		ident, _ = crypto.InitIdentity(mgr.GetKeystorePath(activeID))
	}

	// 2. Storage & State
	storage.InitDB(mgr.GetAccountDir(mgr.ActivePeerID))
	state := core.NewNodeState(ident)

	contacts, _ := storage.GetAllContacts()
	state.Mu.Lock()
	state.PeerAliases = contacts
	state.Mu.Unlock()
	state.DisplayName = func() string {
		acc := mgr.GetAccount(mgr.ActivePeerID)
		if acc == nil { return "" }
		return acc.Name
	}

	// 3. API & Transport
	uiContent, _ := fs.Sub(uiStatic, "ui_dist")
	server := api.NewServer(state, uiContent, mgr)
	go transport.StartServer(state, server.Broadcast)

	// 4. Safe Discovery for Android
	if ifaceName != "" {
		log.Printf("[MOBILE] Starting discovery on: %s", ifaceName)
		go discovery.Start(state, ifaceName)
	}

	// 5. Handshake logic
	go func() {
		for ip := range state.NewPeerChan {
			log.Printf("[BOOT] Handshake with %s", ip)
			state.Mu.RLock()
			h := transport.HandshakePayload{
				PeerID:       state.Identity.PeerID,
				PublicKey:    state.Identity.PublicKey,
				Name:         state.DisplayName(),
				EphemeralPub: hex.EncodeToString(state.EphemeralPub),
			}
			state.Mu.RUnlock()
			transport.SendPacket(state, server.Broadcast, "", ip, "handshake", h)
		}
	}()

	server.Start("8080")
}

func OpenBrowser(url string) {
	log.Printf("[MOBILE] Browser request: %s", url)
}