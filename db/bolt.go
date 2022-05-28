package db

import (
	"encoding/json"
	"fmt"
	"path"

	"go.etcd.io/bbolt"
)

type (
	Bolt struct {
		db *bbolt.DB
	}

	Status struct {
		ID       []byte
		Failed   bool
		Notified bool
	}
)

var (
	statusBucket = []byte("status")
)

func NewBolt(filename string) (*Bolt, error) {
	dbPath := path.Join(filename)

	db, err := bbolt.Open(dbPath, 0600, nil)
	if err != nil {
		return nil, fmt.Errorf("bolot init: %w", err)
	}

	return &Bolt{db}, nil
}

func (b *Bolt) Status(id []byte) (st Status, err error) {
	return st, b.db.Update(func(tx *bbolt.Tx) error {
		bkt, err := tx.CreateBucketIfNotExists(statusBucket)
		if err != nil {
			return err
		}
		v := bkt.Get(id)
		if len(v) == 0 {
			return nil
		}
		return json.Unmarshal(v, &st)
	})
}

func (b *Bolt) SetStatus(id []byte, st Status) error {
	return b.db.Update(func(tx *bbolt.Tx) error {
		bkt, err := tx.CreateBucketIfNotExists(statusBucket)
		if err != nil {
			return err
		}
		v, err := json.Marshal(st)
		if err != nil {
			return err
		}
		return bkt.Put(id, v)
	})
}
