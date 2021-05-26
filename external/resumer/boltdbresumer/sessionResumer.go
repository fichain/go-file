package boltdbresumer

import (
	"encoding/json"
	"errors"
	"fmt"
	"go.etcd.io/bbolt"
)

type SessionSpec struct {
	UserPrivk			[]byte
	TorrentIds 			[]string
}

// Resumer contains methods for saving/loading resume information of a torrent to a BoltDB database.
type SessionResumer struct {
	db     *bbolt.DB
	bucket []byte
}

// New returns a new Resumer.
func NewSessionResumer(db *bbolt.DB, bucket []byte) (*SessionResumer, error) {
	err := db.Update(func(tx *bbolt.Tx) error {
		_, err2 := tx.CreateBucketIfNotExists(bucket)
		return err2
	})
	if err != nil {
		return nil, err
	}
	return &SessionResumer{
		db:     db,
		bucket: bucket,
	}, nil
}

func (r *SessionResumer)Read(user string) (spec *SessionSpec, err error) {
	err = r.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(r.bucket).Bucket([]byte(user))
		if b == nil {
			return nil
		}

		value := b.Get(Keys.UserPrivk)
		if value == nil {
			return fmt.Errorf("key not found: %q", string(Keys.UserPrivk))
		}
		spec = new(SessionSpec)
		spec.UserPrivk = value

		value = b.Get(Keys.TorrentIds)
		if value == nil {
			return fmt.Errorf("key not found: %q", string(Keys.TorrentIds))
		}
		err = json.Unmarshal(value, spec.TorrentIds)
		if err != nil {
			return errors.New("torrent ids style not correct")
		}

		return nil
	})

	return
}

func (r *SessionResumer)Write(user string, spec *SessionSpec) error {
	if spec == nil {
		return errors.New("no session spec")
	}

	torrentIdsByte, err := json.Marshal(spec.TorrentIds)
	if err != nil {
		return err
	}

	return r.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.Bucket(r.bucket).CreateBucketIfNotExists([]byte(user))
		if err != nil {
			return err
		}
		_ = b.Put(Keys.UserPrivk, spec.UserPrivk)
		_ = b.Put(Keys.TorrentIds, torrentIdsByte)
		return nil
	})
}