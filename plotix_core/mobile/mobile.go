// Package mobile provides the entry point for gomobile binding.
// It exposes a simple API for Android (.aar) and iOS (.xcframework).
package mobile

import (
	"encoding/hex"
	"fmt"
	"log"

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

var (
	nodeState  *core.NodeState
	apiServer  *api.Server
	accountMgr *accounts.Manager
)

// StartNode launches the Plotix P2P core.
// dataDir must be the app's internal storage path (e.g. context.filesDir on Android).
// Returns "started" on success or an error string prefixed with "error:".
func StartNode(dataDir string) string {
	ifaces, err := utils.GetInterfacesList()
	if err != nil || len(ifaces) == 0 {
		return "error: no network interfaces available"
	}
	selectedIface := ifaces[0].Name

	accountMgr = accounts.NewManager(dataDir)
	if err := accountMgr.Load(); err != nil {
		return "error: account manager: " + err.Error()
	}

	var ident *crypto.Identity
	if len(accountMgr.Accounts) == 0 {
		var createErr error
		_, ident, createErr = accountMgr.CreateAccount("Mobile User")
		if createErr != nil {
			return "error: create account: " + createErr.Error()
		}
		accountMgr.ActivePeerID = ident.PeerID
		accountMgr.Save()
	} else {
		activeID := accountMgr.LoadActive()
		if activeID == "" || !accountMgr.HasAccount(activeID) {
			activeID = accountMgr.Accounts[0].PeerID
		}
		accountMgr.ActivePeerID = activeID

		var loadErr error
		ident, loadErr = crypto.InitIdentity(accountMgr.GetKeystorePath(activeID))
		if loadErr != nil {
			return "error: identity: " + loadErr.Error()
		}
	}

	log.Printf("[MOBILE] PeerID: %s", ident.PeerID)

	storage.InitDB(accountMgr.GetAccountDir(accountMgr.ActivePeerID))

	nodeState = core.NewNodeState(ident)

	contacts, _ := storage.GetAllContacts()
	nodeState.Mu.Lock()
	nodeState.PeerAliases = contacts
	nodeState.Mu.Unlock()
	nodeState.DisplayName = func() string {
		acc := accountMgr.GetAccount(accountMgr.ActivePeerID)
		if acc == nil || acc.Ghost {
			return ""
		}
		return acc.Name
	}

	apiServer = api.NewServer(nodeState, nil, accountMgr)

	apiServer.SwitchAccount = func(peerID string) error {
		if !accountMgr.HasAccount(peerID) {
			return fmt.Errorf("account not found: %s", peerID)
		}
		nodeState.ResetConnections()
		storage.CloseDB()

		newIdent, err := crypto.InitIdentity(accountMgr.GetKeystorePath(peerID))
		if err != nil {
			return err
		}

		nodeState.Mu.Lock()
		nodeState.Identity = newIdent
		nodeState.Mu.Unlock()

		storage.InitDB(accountMgr.GetAccountDir(peerID))

		contacts, _ := storage.GetAllContacts()
		nodeState.Mu.Lock()
		nodeState.PeerAliases = contacts
		nodeState.Mu.Unlock()

		accountMgr.ActivePeerID = peerID
		accountMgr.Save()

		apiServer.Broadcast <- models.WSEvent{
			Type:    "account_switched",
			Payload: map[string]string{"peer_id": peerID},
		}

		log.Printf("[MOBILE] Switched to account %s", peerID)
		return nil
	}

	go apiServer.Start("8080")
	go transport.StartServer(nodeState, apiServer.Broadcast)
	discovery.Start(nodeState, selectedIface)

	go func() {
		for ip := range nodeState.NewPeerChan {
			log.Printf("[MOBILE] Handshake with %s", ip)
			var displayName string
			if nodeState.DisplayName != nil {
				displayName = nodeState.DisplayName()
			}
			nodeState.Mu.RLock()
			h := transport.HandshakePayload{
				PeerID:       nodeState.Identity.PeerID,
				PublicKey:    nodeState.Identity.PublicKey,
				Name:         displayName,
				EphemeralPub: hex.EncodeToString(nodeState.EphemeralPub),
			}
			nodeState.Mu.RUnlock()
			if err := transport.SendPacket(nodeState, apiServer.Broadcast, "", ip, "handshake", h); err != nil {
				log.Printf("[MOBILE] Handshake error with %s: %v", ip, err)
			}
		}
	}()

	return "started"
}

// GetPeerID returns the current node's PeerID.
func GetPeerID() string {
	if nodeState == nil {
		return ""
	}
	nodeState.Mu.RLock()
	defer nodeState.Mu.RUnlock()
	return nodeState.Identity.PeerID
}

// GetAPIPort returns the local API port the node is listening on.
func GetAPIPort() string {
	return "8080"
}
