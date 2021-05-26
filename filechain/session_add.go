package filechain

import (
	"encoding/hex"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/fichain/go-file/internal/bitfield"
	"github.com/fichain/go-file/internal/magnet"
	"github.com/fichain/go-file/internal/metainfo"
	"github.com/fichain/go-file/internal/storage/filestorage"

	"github.com/fichain/go-file/external/resumer"
)

// AddTorrentOptions contains options for adding a new torrent.
type AddTorrentOptions struct {
	// ID uniquely identifies the torrent in Session.
	// If empty, a random ID is generated.
	ID string
	// Do not start torrent automatically after adding.
	Stopped bool
	// Stop torrent after all pieces are downloaded.
	StopAfterDownload bool
	//data dir
	DataDir string
}

func (s *Session) AddFileId(uri string, opt *AddTorrentOptions) (*torrent, error)  {
	uri = filterOutControlChars(uri)
	if opt == nil {
		opt = &AddTorrentOptions{}
	}

	return s.addFileId(uri, opt)
}

func filterOutControlChars(s string) string {
	var sb strings.Builder
	sb.Grow(len(s))
	for i := 0; i < len(s); i++ {
		b := s[i]
		if b < ' ' || b == 0x7f {
			continue
		}
		sb.WriteByte(b)
	}
	return sb.String()
}

func (s *Session) addFileId(link string, opt *AddTorrentOptions) (*torrent, error)  {
	ma, err := magnet.New(link)
	if err != nil {
		return nil, newInputError(err)
	}
	ma.Peers = []string{}
	ma.Trackers = [][]string{}

	opt.ID = ma.InfoString()
	if opt.DataDir == "" {
		opt.DataDir = s.config.DataDir
	}

	t, err := newTorrent2(
		s,
		time.Now(),
		ma.InfoHash[:],
		ma.Name,
		nil, // info
		nil, // bitfield
		resumer.Stats{},
		//nil, // webseedSources
		opt.StopAfterDownload,
		opt.DataDir,
	)
	if err != nil {
		return nil, err
	}
	go s.checkTorrent(t)
	defer func() {
		if err != nil {
			t.Close()
		}
	}()
	//todo resume
	//rspec := &boltdbresumer.Spec{
	//	InfoHash:          ma.InfoHash[:],
	//	Name:              ma.Name,
	//	Trackers:          ma.Trackers,
	//	FixedPeers:        ma.Peers,
	//	AddedAt:           t.addedAt,
	//	StopAfterDownload: opt.StopAfterDownload,
	//}
	//err = s.resumer.Write(id, rspec)
	//if err != nil {
	//	return nil, err
	//}
	t2 := s.insertTorrent(t)
	if !opt.Stopped {
		err = t2.Start()
	}
	return t2, err
}

func (s *Session) generateStorage(id string, dataDir string) (sto *filestorage.FileStorage, err error) {
	var dest string
	if s.config.DataDirIncludesTorrentID {
		dest = filepath.Join(dataDir, id)
	} else {
		dest = dataDir
	}
	sto, err = filestorage.New(dest)
	if err != nil {
		return
	}
	return
}

func (s *Session) insertTorrent(t *torrent) *torrent {
	t.log.Info("added torrent")
	s.mTorrents.Lock()
	defer s.mTorrents.Unlock()
	s.torrents[t.id] = t
	s.sessionSpec.TorrentIds = append(s.sessionSpec.TorrentIds, t.id)
	return t
}

func (s *Session) CreateFile(dataDir string) (*torrent, error) {
	by, err := metainfo.NewInfoBytes("", []string{dataDir}, false, 0, "", s.log)
	if err != nil {
		s.log.Errorln("create info bytes error!", err)
		return nil, err
	}
	info, err := metainfo.NewInfo(by)
	if err != nil {
		s.log.Errorln("create info error!", err)
		return nil, err
	}
	opt := &AddTorrentOptions{Stopped:false,StopAfterDownload: false}
	opt.ID = hex.EncodeToString(info.Hash[:])
	s.log.Infof("create info success, info: %v, %v", info.Files, info.PieceLength)
	//opt.StopAfterDownload = true

	bf := bitfield.New(info.NumPieces)
	for i := uint32(0); i < info.NumPieces; i++ {
		bf.Set(i)
	}

	torrentPath := path.Join(dataDir, "../")
	t, err := newTorrent2(
		s,
		time.Now(),
		info.Hash[:],
		info.Name,
		info, // info
		bf, // bitfield
		resumer.Stats{},
		opt.StopAfterDownload,
		torrentPath,
	)

	if err != nil {
		return nil, err
	}
	go s.checkTorrent(t)
	defer func() {
		if err != nil {
			t.Close()
		}
	}()

	t2 := s.insertTorrent(t)
	if !opt.Stopped {
		err = t2.Start()
	}
	return t2, err
}
