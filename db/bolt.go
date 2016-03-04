package db

import (
	"encoding/binary"
	"errors"
	"fmt"
	"os"

	"github.com/boltdb/bolt"
)

// Error is an error struct for database operations
type Error struct {
	op  string
	err error
}

func (e Error) Error() string {
	return fmt.Sprintf("%s: %s", e.op, e.err)
}

// Bolt is a container for a bolt database
type Bolt struct {
	DB       *bolt.DB
	Filename string
}

// Open opens a bolt database.
func (b *Bolt) Open(name string) error {
	b.Filename = name
	var notExist bool
	// See if the file exists.  If it does not exist, the buckets will need to
	// be created.
	// TODO: what about handling new buckets?  Should we just create if not
	// exist on DB open?
	_, err := os.Stat(name)
	if err != nil {
		if !os.IsNotExist(err) {
			return Error{"check database file", err}
		}
		notExist = true
	}
	b.DB, err = bolt.Open(name, 0600, nil)
	if err != nil {
		return Error{"open database", err}
	}
	if notExist {
		return b.CreateBuckets()
	}
	return nil
}

// CreateBuckets creates the buckets for margo.
func (b *Bolt) CreateBuckets() error {
	for _, v := range Buckets {
		err := b.DB.Update(func(tx *bolt.Tx) error {
			// even though the buckets shouldn't exist, CreateBucketIfNotExists
			// is used just in case.
			_, err := tx.CreateBucketIfNotExists([]byte(v.String()))
			if err != nil {
				return Error{fmt.Sprintf("create bucket %s", v), err}
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// ClientIDs returns all ClientIDs within the database.
// TODO: for now, it is a slice of strings, in the future it should be a
// slice of clients.
func (b *Bolt) ClientIDs() ([]uint32, error) {
	var ids []uint32
	err := b.DB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(Client.String()))
		if b == nil {
			return Error{fmt.Sprintf("get %s bucket", Client), errors.New("does not exist")}
		}
		c := b.Cursor()
		// TODO: ignore the value for now.  Once data gets saved with the
		// client id, use it.
		// clientIDs are uint32, if the key isn't 4 bytes long a panic will occur.
		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			ids = append(ids, uint32(k[0])|uint32(k[1])<<8|uint32(k[2])<<16|uint32(k[3])<<24)
		}
		return nil
	})
	return ids, err
}

// SaveClient saves a client in the client bucket.
// TODO: For now, the ID is also saved as the value.  Change the func sig and
// save data when appropriate.
func (b *Bolt) SaveClient(id uint32) error {
	return b.DB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(Client.String()))
		bid := make([]byte, 4)
		binary.LittleEndian.PutUint32(bid, id)
		err := b.Put(bid, bid)
		if err != nil {
			return Error{fmt.Sprintf("save client %d", id), err}
		}
		return nil
	})

}
