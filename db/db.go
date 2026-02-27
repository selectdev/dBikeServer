package db

import (
	"time"

	badger "github.com/dgraph-io/badger/v4"
)

type DB struct {
	bdb  *badger.DB
	done chan struct{}
}

func Open(path string) (*DB, error) {
	opts := badger.DefaultOptions(path).WithLogger(nil)
	bdb, err := badger.Open(opts)
	if err != nil {
		return nil, err
	}
	d := &DB{bdb: bdb, done: make(chan struct{})}
	go d.gcLoop()
	return d, nil
}

func (d *DB) Close() error {
	close(d.done)
	return d.bdb.Close()
}

func (d *DB) gcLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-d.done:
			return
		case <-ticker.C:
			d.bdb.RunValueLogGC(0.5)
		}
	}
}

func (d *DB) Get(key string) ([]byte, bool, error) {
	var val []byte
	err := d.bdb.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err == badger.ErrKeyNotFound {
			return nil
		}
		if err != nil {
			return err
		}
		val, err = item.ValueCopy(nil)
		return err
	})
	return val, val != nil, err
}

func (d *DB) Set(key string, val []byte) error {
	return d.bdb.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(key), val)
	})
}

func (d *DB) Delete(key string) error {
	return d.bdb.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte(key))
	})
}

// Scan returns all key-value pairs whose key starts with prefix.
func (d *DB) Scan(prefix string) ([][2][]byte, error) {
	var results [][2][]byte
	err := d.bdb.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		it := txn.NewIterator(opts)
		defer it.Close()
		p := []byte(prefix)
		for it.Seek(p); it.ValidForPrefix(p); it.Next() {
			item := it.Item()
			k := item.KeyCopy(nil)
			v, err := item.ValueCopy(nil)
			if err != nil {
				return err
			}
			results = append(results, [2][]byte{k, v})
		}
		return nil
	})
	return results, err
}

// ScanKeys returns all keys with the given prefix (no values fetched).
func (d *DB) ScanKeys(prefix string) ([]string, error) {
	var keys []string
	err := d.bdb.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()
		p := []byte(prefix)
		for it.Seek(p); it.ValidForPrefix(p); it.Next() {
			keys = append(keys, string(it.Item().KeyCopy(nil)))
		}
		return nil
	})
	return keys, err
}

// ScanReverse returns up to limit key-value pairs with the given prefix in reverse (newest first).
func (d *DB) ScanReverse(prefix string, limit int) ([][2][]byte, error) {
	var results [][2][]byte
	err := d.bdb.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.Reverse = true
		it := txn.NewIterator(opts)
		defer it.Close()
		p := []byte(prefix)
		seekKey := append([]byte(prefix), 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF)
		for it.Seek(seekKey); it.ValidForPrefix(p); it.Next() {
			item := it.Item()
			k := item.KeyCopy(nil)
			v, err := item.ValueCopy(nil)
			if err != nil {
				return err
			}
			results = append(results, [2][]byte{k, v})
			if limit > 0 && len(results) >= limit {
				break
			}
		}
		return nil
	})
	return results, err
}
