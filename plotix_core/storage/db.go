package storage

import (
	"encoding/json"
	"log"
	"path/filepath"

	"go.etcd.io/bbolt"
)

var db *bbolt.DB

// MessageEntity - структура для хранения в БД
type MessageEntity struct {
	ID        string   `json:"id"`
	Parents   []string `json:"parents"`
	Sender    string   `json:"sender"`
	Text      string   `json:"text"`
	Timestamp int64    `json:"timestamp"`
}

func InitDB(path string) {
	var err error
	dbPath := filepath.Join(path, "plotix.db")
	db, err = bbolt.Open(dbPath, 0600, nil)
	if err != nil {
		log.Fatal(err)
	}

	db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("messages"))
		return err
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
		return peerBucket.Put([]byte(msg.ID), data)
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
	return history, err
}

func CloseDB() {
	if db != nil {
		db.Close()
	}
}
