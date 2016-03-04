package db

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/boltdb/bolt"
)

func TestBoltDB(t *testing.T) {
	var db Bolt
	tmpDir, err := ioutil.TempDir("", "autofact")
	if err != nil {
		t.Errorf("error creating tmpDir for db: %s", err)
		return
	}
	// open the db
	err = db.Open(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Errorf("error opening db file %s: %s", filepath.Join(tmpDir, "test.db"), err)
		return
	}
	defer os.RemoveAll(tmpDir)
	defer db.DB.Close()

	// check that the buckets were Created
	for _, v := range Buckets {
		db.DB.View(func(tx *bolt.Tx) error {
			c := tx.Bucket([]byte(v.String()))
			if c == nil {
				t.Errorf("expected %s to exist; got nil", v)
			}
			return nil
		})
	}

	// Add clients to the bucket
	IDs := []uint32{1, 11, 42}
	for _, v := range IDs {
		err := db.SaveClient(v)
		if err != nil {
			t.Errorf("expected no error; got %s", err)
			return
		}
	}

	// get clients
	ids, err := db.ClientIDs()
	if err != nil {
		t.Errorf("expected no error; got %s", err)
		return
	}
	if len(ids) != len(IDs) {
		t.Errorf("clientID: expected %d elements, got %d", len(IDs), len(ids))
		return
	}
	for _, v := range IDs {
		var found bool
		for _, vv := range ids {
			if vv == v {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected %d; not found", v)
			return
		}
	}
}
