// Package boltdbresumer provides a Resumer implementation that uses a Bolt database file as storage.
package boltdbresumer

import (
	"encoding/json"
	"fmt"
	"runtime/debug"
	"strconv"
	"time"

	"go.etcd.io/bbolt"
)

// Keys for the persisten storage.
var Keys = struct {
	InfoHash        []byte
	Name            []byte
	FixedPeers      []byte
	Dest            []byte
	Info            []byte
	Bitfield        []byte
	AddedAt         []byte
	BytesDownloaded []byte
	BytesUploaded   []byte
	BytesWasted     []byte
	SeededFor       []byte
	Started         []byte

	//add
	DataDir 		[]byte

	//session
	UserPrivk		[]byte
	TorrentIds		[]byte
}{
	InfoHash:        []byte("info_hash"),
	Name:            []byte("name"),
	FixedPeers:      []byte("fixed_peers"),
	Dest:            []byte("dest"),
	Info:            []byte("info"),
	Bitfield:        []byte("bitfield"),
	AddedAt:         []byte("added_at"),
	BytesDownloaded: []byte("bytes_downloaded"),
	BytesUploaded:   []byte("bytes_uploaded"),
	BytesWasted:     []byte("bytes_wasted"),
	SeededFor:       []byte("seeded_for"),
	Started:         []byte("started"),

	//add
	DataDir: 		 []byte("data_dir"),

	//session
	UserPrivk:		 []byte("user_privk"),
	TorrentIds:		 []byte("torrentIds"),
}

// Resumer contains methods for saving/loading resume information of a torrent to a BoltDB database.
type TorrentResumer struct {
	db     		*bbolt.DB
	bucket 		[]byte
	user  []byte
}

// New returns a new Resumer.
func NewTorrentResumer(db *bbolt.DB, bucket []byte, user []byte) (*TorrentResumer, error) {
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
	return &TorrentResumer{
		db:     db,
		bucket: bucket,
		user: 	user,
	}, nil
}

// Write the torrent spec for torrent with `torrentID`.
func (r *TorrentResumer) Write(torrentID string, spec *Spec) error {
	fixedPeers, err := json.Marshal(spec.FixedPeers)
	if err != nil {
		return err
	}
	return r.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.Bucket(r.user).Bucket(r.bucket).CreateBucketIfNotExists([]byte(torrentID))
		if err != nil {
			return err
		}
		_ = b.Put(Keys.InfoHash, spec.InfoHash)
		_ = b.Put(Keys.Name, []byte(spec.Name))
		_ = b.Put(Keys.FixedPeers, fixedPeers)
		_ = b.Put(Keys.Info, spec.Info)
		_ = b.Put(Keys.Bitfield, spec.Bitfield)
		_ = b.Put(Keys.AddedAt, []byte(spec.AddedAt.Format(time.RFC3339)))
		_ = b.Put(Keys.BytesDownloaded, []byte(strconv.FormatInt(spec.BytesDownloaded, 10)))
		_ = b.Put(Keys.BytesUploaded, []byte(strconv.FormatInt(spec.BytesUploaded, 10)))
		_ = b.Put(Keys.BytesWasted, []byte(strconv.FormatInt(spec.BytesWasted, 10)))
		_ = b.Put(Keys.SeededFor, []byte(spec.SeededFor.String()))
		_ = b.Put(Keys.Started, []byte(strconv.FormatBool(spec.Started)))
		_ = b.Put(Keys.DataDir, []byte(spec.DataDir))
		return nil
	})
}

// WriteInfo writes only the info dict of a torrent.
func (r *TorrentResumer) WriteInfo(torrentID string, value []byte) error {
	return r.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(r.user).Bucket(r.bucket).Bucket([]byte(torrentID))
		if b == nil {
			return nil
		}
		return b.Put(Keys.Info, value)
	})
}

// WriteBitfield writes only bitfield of a torrent.
func (r *TorrentResumer) WriteBitfield(torrentID string, value []byte) error {
	return r.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(r.user).Bucket(r.bucket).Bucket([]byte(torrentID))
		if b == nil {
			return nil
		}
		return b.Put(Keys.Bitfield, value)
	})
}

// WriteStarted writes the start status of a torrent.
func (r *TorrentResumer) WriteStarted(torrentID string, value bool) error {
	return r.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(r.user).Bucket(r.bucket).Bucket([]byte(torrentID))
		if b == nil {
			return nil
		}
		return b.Put(Keys.Started, []byte(strconv.FormatBool(value)))
	})
}

func (r *TorrentResumer) Read(torrentID string) (spec *Spec, err error) {
	defer debug.SetPanicOnFault(debug.SetPanicOnFault(true))
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("cannot read torrent %q from db: %s", torrentID, r)
		}
	}()
	err = r.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(r.user).Bucket(r.bucket).Bucket([]byte(torrentID))
		if b == nil {
			return fmt.Errorf("bucket not found: %q", torrentID)
		}

		value := b.Get(Keys.InfoHash)
		if value == nil {
			return fmt.Errorf("key not found: %q", string(Keys.InfoHash))
		}

		spec = new(Spec)
		spec.InfoHash = make([]byte, len(value))
		copy(spec.InfoHash, value)

		var err error

		value = b.Get(Keys.Name)
		if value != nil {
			spec.Name = string(value)
		}

		value = b.Get(Keys.FixedPeers)
		if value != nil {
			err = json.Unmarshal(value, &spec.FixedPeers)
			if err != nil {
				return err
			}
		}

		value = b.Get(Keys.Info)
		if value != nil {
			spec.Info = make([]byte, len(value))
			copy(spec.Info, value)
		}

		value = b.Get(Keys.Bitfield)
		if value != nil {
			spec.Bitfield = make([]byte, len(value))
			copy(spec.Bitfield, value)
		}

		value = b.Get(Keys.AddedAt)
		if value != nil {
			spec.AddedAt, err = time.Parse(time.RFC3339, string(value))
			if err != nil {
				return err
			}
		}

		value = b.Get(Keys.BytesDownloaded)
		if value != nil {
			spec.BytesDownloaded, err = strconv.ParseInt(string(value), 10, 64)
			if err != nil {
				return err
			}
		}

		value = b.Get(Keys.BytesUploaded)
		if value != nil {
			spec.BytesUploaded, err = strconv.ParseInt(string(value), 10, 64)
			if err != nil {
				return err
			}
		}

		value = b.Get(Keys.BytesWasted)
		if value != nil {
			spec.BytesWasted, err = strconv.ParseInt(string(value), 10, 64)
			if err != nil {
				return err
			}
		}

		value = b.Get(Keys.SeededFor)
		if value != nil {
			spec.SeededFor, err = time.ParseDuration(string(value))
			if err != nil {
				return err
			}
		}

		value = b.Get(Keys.Started)
		if value != nil {
			spec.Started, err = strconv.ParseBool(string(value))
			if err != nil {
				return err
			}
		}

		value = b.Get(Keys.DataDir)
		if value != nil {
			spec.DataDir = string(value)
		}

		return nil
	})
	return
}

func (r *TorrentResumer)Del() error {
	return r.db.Update(func(tx *bbolt.Tx) error {
		return tx.Bucket(r.user).DeleteBucket(r.bucket)
	})
}
