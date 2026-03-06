package storage

import (
	"encoding/json"
	"log"
	"path/filepath"
	"sort"

	"go.etcd.io/bbolt"
)

var db *bbolt.DB

type MessageEntity struct {
	ID        string   `json:"id"`
	Parents   []string `json:"parents"`
	Sender    string   `json:"sender"`
	Text      string   `json:"text"`
	Timestamp int64    `json:"timestamp"`
	Delivered bool     `json:"delivered"`
}

func InitDB(path string) {
	CloseDB()

	var err error
	dbPath := filepath.Join(path, "plotix.db")
	db, err = bbolt.Open(dbPath, 0600, nil)
	if err != nil {
		log.Fatal(err)
	}

	db.Update(func(tx *bbolt.Tx) error {
		_, _ = tx.CreateBucketIfNotExists([]byte("messages"))
		_, _ = tx.CreateBucketIfNotExists([]byte("contacts"))
		_, _ = tx.CreateBucketIfNotExists([]byte("heads"))
		return nil
	})

	log.Printf("[STORAGE] Database opened: %s", dbPath)
}

func SaveMessage(peerID string, msg MessageEntity) error {
	return db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("messages"))
		peerBucket, err := b.CreateBucketIfNotExists([]byte(peerID))
		if err != nil {
			return err
		}
		data, _ := json.Marshal(msg)
		if err := peerBucket.Put([]byte(msg.ID), data); err != nil {
			return err
		}

		h := tx.Bucket([]byte("heads"))
		var currentHeads []string
		headData := h.Get([]byte(peerID))
		if headData != nil {
			json.Unmarshal(headData, &currentHeads)
		}

		newHeads := []string{msg.ID}
		for _, head := range currentHeads {
			isParent := false
			for _, parent := range msg.Parents {
				if head == parent {
					isParent = true
					break
				}
			}
			if !isParent {
				newHeads = append(newHeads, head)
			}
		}

		newHeadData, _ := json.Marshal(newHeads)
		return h.Put([]byte(peerID), newHeadData)
	})
}

func GetHistory(peerID string) ([]MessageEntity, error) {
	var history []MessageEntity
	err := db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("messages"))
		peerBucket := b.Bucket([]byte(peerID))
		if peerBucket == nil {
			return nil
		}

		return peerBucket.ForEach(func(k, v []byte) error {
			var m MessageEntity
			json.Unmarshal(v, &m)
			history = append(history, m)
			return nil
		})
	})

	sort.Slice(history, func(i, j int) bool {
		return history[i].Timestamp < history[j].Timestamp
	})

	return history, err
}

func GetHeads(peerID string) []string {
	var heads []string
	db.View(func(tx *bbolt.Tx) error {
		h := tx.Bucket([]byte("heads"))
		data := h.Get([]byte(peerID))
		if data != nil {
			json.Unmarshal(data, &heads)
		}
		return nil
	})
	return heads
}

func MarkDelivered(peerID, msgID string) error {
	return db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("messages"))
		peerBucket := b.Bucket([]byte(peerID))
		if peerBucket == nil {
			return nil
		}

		val := peerBucket.Get([]byte(msgID))
		if val == nil {
			return nil
		}

		var msg MessageEntity
		json.Unmarshal(val, &msg)
		msg.Delivered = true

		data, _ := json.Marshal(msg)
		return peerBucket.Put([]byte(msgID), data)
	})
}

func GetPendingMessages(peerID string) ([]MessageEntity, error) {
	var pending []MessageEntity
	err := db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("messages"))
		peerBucket := b.Bucket([]byte(peerID))
		if peerBucket == nil {
			return nil
		}

		return peerBucket.ForEach(func(k, v []byte) error {
			var m MessageEntity
			json.Unmarshal(v, &m)
			if !m.Delivered && m.Sender != peerID {
				pending = append(pending, m)
			}
			return nil
		})
	})
	return pending, err
}

func SaveContact(peerID, name string) error {
	return db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("contacts"))
		if name == "" {
			return b.Delete([]byte(peerID))
		}
		return b.Put([]byte(peerID), []byte(name))
	})
}

func GetAllContacts() (map[string]string, error) {
	contacts := make(map[string]string)
	err := db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("contacts"))
		if b == nil {
			return nil
		}
		return b.ForEach(func(k, v []byte) error {
			contacts[string(k)] = string(v)
			return nil
		})
	})
	return contacts, err
}

func MessageExists(peerID, msgID string) bool {
	exists := false
	db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("messages"))
		if b == nil {
			return nil
		}
		peerBucket := b.Bucket([]byte(peerID))
		if peerBucket == nil {
			return nil
		}
		if peerBucket.Get([]byte(msgID)) != nil {
			exists = true
		}
		return nil
	})
	return exists
}

func GetKnownPeers() ([]string, error) {
	var peers []string
	err := db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("messages"))
		if b == nil {
			return nil
		}
		return b.ForEach(func(k, v []byte) error {
			if v == nil {
				peers = append(peers, string(k))
			}
			return nil
		})
	})
	return peers, err
}

func CloseDB() {
	if db != nil {
		db.Close()
	}
}
