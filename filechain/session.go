package filechain

import (
	"errors"
	p2p2 "github.com/fichain/go-file/external/p2p"
	"github.com/fichain/go-file/external/resumer/boltdbresumer"
	"github.com/fichain/go-file/internal/piececache"
	"github.com/fichain/go-file/internal/resourcemanager"
	"github.com/fichain/go-file/internal/semaphore"
	"github.com/juju/ratelimit"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fichain/go-file/internal/blocklist"
	"github.com/fichain/go-file/internal/logger"
	"github.com/libp2p/go-libp2p-core/host"
	discovery "github.com/libp2p/go-libp2p-discovery"
	"github.com/mitchellh/go-homedir"
	"go.etcd.io/bbolt"
)

var (
	sessionBucket         = []byte("session")
	torrentsBucket        = []byte("torrents")
	blocklistKey          = []byte("blocklist")
	blocklistTimestampKey = []byte("blocklist-timestamp")
	blocklistURLHashKey   = []byte("blocklist-url-hash")
)

// Session contains torrents, DHT node, caches and other data structures shared by multiple torrents.
type Session struct {
	host			host.Host
	routeDiscovery   *discovery.RoutingDiscovery	//dht
	log            logger.Logger

	config         Config
	createdAt      time.Time

	mPeerRequests   sync.Mutex
	mTorrents          sync.RWMutex
	torrents           map[string]*torrent
	ram            *resourcemanager.ResourceManager

	sessionSpec 	*boltdbresumer.SessionSpec

	//todo ratelimit
	bucketDownload *ratelimit.Bucket
	bucketUpload   *ratelimit.Bucket

	metrics        *sessionMetrics

	pieceCache     *piececache.Cache

	semWrite       *semaphore.Semaphore

	db             *bbolt.DB
	sessionResumer        *boltdbresumer.SessionResumer
	resumer				  *boltdbresumer.TorrentResumer
}

// NewSession creates a new Session for downloading and seeding torrents.
// Returned session must be closed after use.
func NewSession(cfg Config) (*Session, error) {
	if cfg.MaxOpenFiles > 0 {
		err := setNoFile(cfg.MaxOpenFiles)
		if err != nil {
			return nil, errors.New("cannot change max open files limit: " + err.Error())
		}
	}
	var err error
	if len(cfg.LibP2pUser) == 0 {
		return nil, errors.New("no p2p user")
	}

	cfg.Database, err = homedir.Expand(cfg.Database)
	if err != nil {
		return nil, err
	}
	cfg.DataDir, err = homedir.Expand(cfg.DataDir)
	if err != nil {
		return nil, err
	}
	err = os.MkdirAll(filepath.Dir(cfg.Database), 0750)
	if err != nil {
		return nil, err
	}
	l := logger.New("session")
	if cfg.Debug {
		logger.SetDebug()
	}

	db, err := bbolt.Open(cfg.Database, 0640, &bbolt.Options{Timeout: time.Second})
	if err == bbolt.ErrTimeout {
		return nil, errors.New("resume database is locked by another process")
	} else if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			db.Close()
		}
	}()
	//resumer
	sessionRe, err := boltdbresumer.NewSessionResumer(db, sessionBucket)
	if err != nil {
		return nil, err
	}

	torrentRe, err := boltdbresumer.NewTorrentResumer(db, torrentsBucket)
	if err != nil {
		return nil, err
	}

	//sessionspec
	sessionSpec, err := sessionRe.Read(cfg.LibP2pUser)
	if err != nil {
		return nil, err
	}
	if sessionSpec == nil {
		sessionSpec = new(boltdbresumer.SessionSpec)
	}

	bl := blocklist.New()
	bl.Logger = l.Errorf
	c := &Session{
		//host: host,
		//routeDiscovery: routeDiscovery,
		log:                l,

		config:             cfg,
		torrents:           make(map[string]*torrent),
		createdAt:          time.Now(),
		ram:                resourcemanager.New(cfg.WriteCacheSize),
		//metrics:
		pieceCache:         piececache.New(cfg.ReadCacheSize, cfg.ReadCacheTTL, cfg.ParallelReads),
		semWrite:           semaphore.New(int(cfg.ParallelWrites)),

		db: 				db,
		sessionResumer: 	sessionRe,
		resumer:		torrentRe,
		sessionSpec: 		sessionSpec,
	}

	//host
	priv, err := c.getCurrentUserKey(c.sessionSpec)
	if err != nil {
		return nil, err
	}

	host, err := p2p2.NewRoutedHost(cfg.LibP2pPort, cfg.LibP2pBootStrap, priv)
	if err != nil {
		return nil, err
	}
	l.Infof("create host success!, id is: %v, addrs is: %v\n", host.ID(), host.Addrs())
	routeDiscovery, err := p2p2.NewRoutedDiscovery(host)
	if err != nil {
		return nil, err
	}
	c.host = host
	c.routeDiscovery = routeDiscovery

	l.Infoln("create route discovery success!")

	//todo blocklist for libp2p

	//todo init metrics
	c.initMetrics()

	c.loadExistingTorrents(sessionSpec.TorrentIds)

	//todo update stats
	//go c.updateStatsLoop()

	return c, nil
}

// Close stops all torrents and release the resources.
func (s *Session) Close() error {
	var wg sync.WaitGroup
	s.mTorrents.Lock()
	wg.Add(len(s.torrents))
	for _, t := range s.torrents {
		go func(t *torrent) {
			//t.Close()
			wg.Done()
		}(t)
	}
	wg.Wait()
	s.torrents = nil
	s.mTorrents.Unlock()
	return nil
}
