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
	user   []byte
}

// New returns a new Resumer.
func NewSessionResumer(db *bbolt.DB, bucket []byte, user []byte) (*SessionResumer, error) {
	err := db.Update(func(tx *bbolt.Tx) error {
		b, err2 := tx.CreateBucketIfNotExists(user)
		if err2 != nil {
			return err2
		}
		_, err3 := b.CreateBucketIfNotExists(bucket)
		return err3
	})
	if err != nil {
		return nil, err
	}
	return &SessionResumer{
		db:     db,
		bucket: bucket,
		user:	user,
	}, nil
}

func (r *SessionResumer)Read() (spec *SessionSpec, err error) {
	spec = new(SessionSpec)
	spec.TorrentIds = []string{}

	err = r.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(r.user).Bucket(r.bucket)
		if b == nil {
			return nil
		}

		value := b.Get(Keys.UserPrivk)
		if value == nil {
			return fmt.Errorf("key not found: %q", string(Keys.UserPrivk))
		}
		spec.UserPrivk = value

		value = b.Get(Keys.TorrentIds)
		if value == nil {
			return fmt.Errorf("key not found: %q", string(Keys.TorrentIds))
		}
		err = json.Unmarshal(value, &spec.TorrentIds)
		if err != nil {
			return err
		}

		return nil
	})

	return
}

func (r *SessionResumer)Write(spec *SessionSpec) error {
	if spec == nil {
		return errors.New("no session spec")
	}

	torrentIdsByte, err := json.Marshal(spec.TorrentIds)
	if err != nil {
		return err
	}

	return r.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(r.user).Bucket(r.bucket)
		_ = b.Put(Keys.UserPrivk, spec.UserPrivk)
		_ = b.Put(Keys.TorrentIds, torrentIdsByte)
		return nil
	})
}

func (r *SessionResumer)WriteTorrentIds(torrentIds []string) error {
	torrentIdsByte, err := json.Marshal(torrentIds)
	if err != nil {
		return err
	}

	return r.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(r.user).Bucket(r.bucket)
		_ = b.Put(Keys.TorrentIds, torrentIdsByte)
		return nil
	})
}

func (r *SessionResumer)Del() error {
	return r.db.Update(func(tx *bbolt.Tx) error {
		return tx.Bucket(r.user).DeleteBucket(r.bucket)
	})
}