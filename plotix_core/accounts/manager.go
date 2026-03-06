package accounts

import (
	"encoding/json"
	"os"
	"path/filepath"

	"plotix_core/crypto"
)

type AccountInfo struct {
	PeerID string `json:"peer_id"`
	Name   string `json:"name"`
	Ghost  bool   `json:"ghost"`
}

type Manager struct {
	DataDir      string
	Accounts     []AccountInfo
	ActivePeerID string
}

func NewManager(dataDir string) *Manager {
	return &Manager{
		DataDir: dataDir,
	}
}

func (m *Manager) Load() error {
	os.MkdirAll(m.DataDir, 0755)

	acPath := filepath.Join(m.DataDir, "accounts.json")
	data, err := os.ReadFile(acPath)
	if err != nil {
		if os.IsNotExist(err) {
			m.Accounts = []AccountInfo{}
			return nil
		}
		return err
	}

	return json.Unmarshal(data, &m.Accounts)
}

func (m *Manager) Save() error {
	os.MkdirAll(m.DataDir, 0755)

	data, err := json.MarshalIndent(m.Accounts, "", "  ")
	if err != nil {
		return err
	}

	acPath := filepath.Join(m.DataDir, "accounts.json")
	if err := os.WriteFile(acPath, data, 0600); err != nil {
		return err
	}

	activePath := filepath.Join(m.DataDir, "active.txt")
	return os.WriteFile(activePath, []byte(m.ActivePeerID), 0600)
}

func (m *Manager) LoadActive() string {
	activePath := filepath.Join(m.DataDir, "active.txt")
	data, err := os.ReadFile(activePath)
	if err != nil {
		return ""
	}
	return string(data)
}

func (m *Manager) CreateAccount(name string) (*AccountInfo, *crypto.Identity, error) {
	keystorePath := filepath.Join(m.DataDir, "temp_keystore.json")
	ident, err := crypto.GenerateAndSave(keystorePath)
	if err != nil {
		return nil, nil, err
	}

	accountDir := filepath.Join(m.DataDir, ident.PeerID)
	os.MkdirAll(accountDir, 0755)

	finalPath := filepath.Join(accountDir, "keystore.json")
	if err := os.Rename(keystorePath, finalPath); err != nil {
		return nil, nil, err
	}

	info := AccountInfo{
		PeerID: ident.PeerID,
		Name:   name,
	}
	m.Accounts = append(m.Accounts, info)
	m.Save()

	return &info, ident, nil
}

func (m *Manager) ListAccounts() []AccountInfo {
	return m.Accounts
}

func (m *Manager) GetAccountDir(peerID string) string {
	return filepath.Join(m.DataDir, peerID)
}

func (m *Manager) GetKeystorePath(peerID string) string {
	return filepath.Join(m.DataDir, peerID, "keystore.json")
}

func (m *Manager) GetDBPath(peerID string) string {
	return filepath.Join(m.DataDir, peerID)
}

func (m *Manager) SetName(peerID, name string) bool {
	for i, a := range m.Accounts {
		if a.PeerID == peerID {
			m.Accounts[i].Name = name
			m.Save()
			return true
		}
	}
	return false
}

func (m *Manager) GetAccount(peerID string) *AccountInfo {
	for i, a := range m.Accounts {
		if a.PeerID == peerID {
			return &m.Accounts[i]
		}
	}
	return nil
}

func (m *Manager) SetGhost(peerID string, ghost bool) bool {
	for i, a := range m.Accounts {
		if a.PeerID == peerID {
			m.Accounts[i].Ghost = ghost
			m.Save()
			return true
		}
	}
	return false
}

func (m *Manager) HasAccount(peerID string) bool {
	for _, a := range m.Accounts {
		if a.PeerID == peerID {
			return true
		}
	}
	return false
}

func (m *Manager) MigrateExisting(rootDir string) (*crypto.Identity, error) {
	keystoreSrc := filepath.Join(rootDir, "keystore.json")
	dbSrc := filepath.Join(rootDir, "plotix.db")

	ident, err := crypto.LoadIdentity(keystoreSrc)
	if err != nil {
		return nil, err
	}

	accountDir := filepath.Join(m.DataDir, ident.PeerID)
	os.MkdirAll(accountDir, 0755)

	keystoreDst := filepath.Join(accountDir, "keystore.json")
	copyFile(keystoreSrc, keystoreDst)

	if _, err := os.Stat(dbSrc); err == nil {
		dbDst := filepath.Join(accountDir, "plotix.db")
		copyFile(dbSrc, dbDst)
	}

	info := AccountInfo{
		PeerID: ident.PeerID,
		Name:   "",
	}
	m.Accounts = append(m.Accounts, info)
	m.ActivePeerID = ident.PeerID
	m.Save()

	return ident, nil
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0600)
}
