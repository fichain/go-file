package filechain

import (
	"encoding/hex"
	"errors"
	"github.com/fichain/go-file/external/resumer"
	"github.com/fichain/go-file/internal/bitfield"
	"github.com/fichain/go-file/internal/metainfo"
)

var errTooManyPieces = errors.New("too many pieces")

func (s *Session) loadExistingTorrents(ids []string) {
	var loaded int
	var started []*torrent
	for _, id := range ids {
		t, hasStarted, err := s.loadExistingTorrent(id)
		if err != nil {
			s.log.Error(err)
			//todo invalid
			//s.invalidTorrentIDs = append(s.invalidTorrentIDs, id)
			continue
		}
		s.log.Infof("loaded existing torrent: #%s %s", id, t.Name())
		loaded++
		if hasStarted {
			started = append(started, t)
		}
	}
	s.log.Infof("loaded %d existing torrents", loaded)
	if s.config.ResumeOnStartup {
		for _, t := range started {
			t.Start()
		}
	}
}

func (s *Session) parseInfo(b []byte) (*metainfo.Info, error) {
	i, err := metainfo.NewInfo(b)
	if err != nil {
		return nil, err
	}
	if i.NumPieces > s.config.MaxPieces {
		return nil, errTooManyPieces
	}
	return i, nil
}

func (s *Session) loadExistingTorrent(id string) (tt *torrent, hasStarted bool, err error) {
	spec, err := s.resumer.Read(id)
	if err != nil {
		return
	}
	s.log.Debugf("current load torrent info:%v, data dir: %v\n", hex.EncodeToString(spec.InfoHash), spec.DataDir)
	hasStarted = spec.Started
	var info *metainfo.Info
	var bf *bitfield.Bitfield
	if len(spec.Info) > 0 {
		info2, err2 := s.parseInfo(spec.Info)
		if err2 != nil {
			return nil, spec.Started, err2
		}
		info = info2
		if len(spec.Bitfield) > 0 {
			bf3, err3 := bitfield.NewBytes(spec.Bitfield, info.NumPieces)
			if err3 != nil {
				return nil, spec.Started, err3
			}
			bf = bf3
		}
	}

	if spec.DataDir == "" {
		return nil,spec.Started, errors.New("torrent no data dir")
	}

	if len(spec.InfoHash) == 0 {
		return nil, spec.Started, errors.New("torrent info hash not exist")
	}

	t, err := newTorrent2(
		s,
		spec.AddedAt,
		spec.InfoHash,
		spec.Name,
		info,
		bf,
		resumer.Stats{
			BytesDownloaded: spec.BytesDownloaded,
			BytesUploaded:   spec.BytesUploaded,
			BytesWasted:     spec.BytesWasted,
			SeededFor:       int64(spec.SeededFor),
		},
		spec.StopAfterDownload,
		spec.DataDir,
		id,
	)
	if err != nil {
		return
	}
	//go s.checkTorrent(t)

	tt = s.insertTorrent(t)
	return
}

// CleanDatabase removes invalid records in the database.
// Normally you don't need to call this.
//func (s *Session) CleanDatabase() error {
//	err := s.db.Update(func(tx *bbolt.Tx) error {
//		b := tx.Bucket(torrentsBucket)
//		for _, id := range s.invalidTorrentIDs {
//			err := b.DeleteBucket([]byte(id))
//			if err != nil {
//				return err
//			}
//		}
//		return nil
//	})
//	if err != nil {
//		return err
//	}
//	s.invalidTorrentIDs = nil
//	return nil
//}

// CompactDatabase rewrites the database using existing torrent records to a new file.
// Normally you don't need to call this.
//func (s *Session) CompactDatabase(output string) error {
//	db, err := bbolt.Open(output, 0600, nil)
//	if err != nil {
//		return err
//	}
//	defer db.Close()
//	err = db.Update(func(tx *bbolt.Tx) error {
//		_, err2 := tx.CreateBucketIfNotExists(torrentsBucket)
//		return err2
//	})
//	if err != nil {
//		return err
//	}
//	res, err := boltdbresumer.New(db, torrentsBucket)
//	if err != nil {
//		return err
//	}
//	for _, t := range s.torrents {
//		spec := &boltdbresumer.Spec{
//			InfoHash:          t.torrent.InfoHash(),
//			Port:              t.torrent.port,
//			Name:              t.torrent.name,
//			Trackers:          t.torrent.rawTrackers,
//			URLList:           t.torrent.rawWebseedSources,
//			FixedPeers:        t.torrent.fixedPeers,
//			Info:              t.torrent.info.Bytes,
//			AddedAt:           t.torrent.addedAt,
//			StopAfterDownload: t.torrent.stopAfterDownload,
//		}
//		err = res.Write(t.torrent.id, spec)
//		if err != nil {
//			return err
//		}
//	}
//	return db.Close()
//}
