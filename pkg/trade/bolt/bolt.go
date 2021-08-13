package bolt

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/boltdb/bolt"
	"github.com/igolaizola/zeken/pkg/trade"
)

func New(path string) (*Store, error) {
	// Open the my.db data file in your current directory.
	// It will be created if it doesn't exist.
	db, err := bolt.Open(path, 0600, nil)
	if err != nil {
		return nil, fmt.Errorf("bolt: couldn't open bold db %s: %w", path, err)
	}
	if err := db.Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists([]byte("trades")); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("bolt: couldn't create bucket: %w", err)
	}
	return &Store{db: db}, nil
}

type Store struct {
	db *bolt.DB
}

func (s *Store) Close() {
	s.db.Close()
}

func (s *Store) List(from time.Time, to time.Time, finished bool) ([]*trade.Trade, error) {
	var trades []*trade.Trade
	if err := s.db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket([]byte("trades")).Cursor()

		// Time range
		min := []byte(from.UTC().Format(time.RFC3339Nano))
		max := []byte(to.UTC().Format(time.RFC3339Nano))

		for k, v := c.Seek(min); k != nil && bytes.Compare(k, max) <= 0; k, v = c.Next() {
			var t trade.Trade
			if err := json.Unmarshal(v, &t); err != nil {
				return fmt.Errorf("couldn't decode: %w", err)
			}
			if t.StartTime.Before(from) {
				continue
			}
			if t.StartTime.After(to) {
				continue
			}
			hasFinished := t.EndTime != time.Time{}
			if hasFinished != finished {
				continue
			}
			trades = append(trades, &t)
		}

		return nil
	}); err != nil {
		return nil, fmt.Errorf("bold: couldn't query: %w", err)
	}
	return trades, nil
}

func (s *Store) Update(t *trade.Trade) error {
	key := []byte(t.StartTime.UTC().Format(time.RFC3339Nano))
	if err := s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("trades"))
		byt, err := json.Marshal(t)
		if err != nil {
			return fmt.Errorf("couldn't encode: %w", err)
		}
		return b.Put([]byte(key), byt)
	}); err != nil {
		return fmt.Errorf("bolt: couldn't put %s: %w", key, err)
	}
	return nil
}

func (s *Store) Delete(t *trade.Trade) error {
	key := []byte(t.StartTime.UTC().Format(time.RFC3339Nano))
	if err := s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("trades"))
		key := []byte(t.StartTime.UTC().Format(time.RFC3339Nano))
		return b.Delete([]byte(key))
	}); err != nil {
		return fmt.Errorf("store: couldn't delete %s: %w", key, err)
	}
	return nil
}
