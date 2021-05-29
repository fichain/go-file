package filechain

import (
	"errors"
	"github.com/fichain/go-file/internal/storage/filestorage"
	"net"
	"sync"
	"time"

	"github.com/fichain/go-file/internal/allocator"
	"github.com/fichain/go-file/internal/announcer"
	"github.com/fichain/go-file/internal/bitfield"
	//"github.com/fichain/go-file/internal/blocklist"
	"github.com/fichain/go-file/internal/bufferpool"
	"github.com/fichain/go-file/internal/externalip"
	"github.com/fichain/go-file/internal/infodownloader"
	"github.com/fichain/go-file/internal/logger"
	"github.com/fichain/go-file/internal/metainfo"
	"github.com/fichain/go-file/internal/pexlist"
	"github.com/fichain/go-file/internal/piece"
	"github.com/fichain/go-file/internal/piecedownloader"
	"github.com/fichain/go-file/internal/piecewriter"
	"github.com/fichain/go-file/internal/storage"
	"github.com/fichain/go-file/internal/suspendchan"
	"github.com/fichain/go-file/internal/unchoker"
	"github.com/fichain/go-file/internal/verifier"
	"github.com/rcrowley/go-metrics"

	"github.com/fichain/go-file/external/addrlist"
	"github.com/fichain/go-file/external/peer"
	"github.com/fichain/go-file/external/piecepicker"
	"github.com/fichain/go-file/external/resumer"
	"github.com/libp2p/go-libp2p-core/network"
	p2pPeer "github.com/libp2p/go-libp2p-core/peer"
)

// torrent connects to peers and downloads files from swarm.
type torrent struct {
	//todo add
	incomingStreamC chan network.Stream
	//mconnectedPeers sync.RWMutex
	connectedPeers 	map[p2pPeer.ID]*peer.Peer
	// Also keep a reference to incoming and outgoing peers separately to count them quickly.
	incomingPeers 	map[p2pPeer.ID]struct{}
	outgoingPeers 	map[p2pPeer.ID]struct{}

	incoimgPeersC 	chan []p2pPeer.AddrInfo

	infoDownloaderPeers       map[p2pPeer.ID]*infodownloader.InfoDownloader
	infoDownloaderPeersSnubbed map[p2pPeer.ID]*infodownloader.InfoDownloader

	mDhtNeedPeer sync.RWMutex
	dhtNeedPeer bool

	dataDir		string
	//use
	// Peers are sent to this channel when they are disconnected.
	peerDisconnectedC chan *peer.Peer
	// Piece messages coming from peers are sent this channel.
	pieceMessagesC *suspendchan.Chan
	// Other messages coming from peers are sent to this channel.
	messages chan peer.Message
	// When a peer has snubbed us, a message sent to this channel.
	peerSnubbedC chan *peer.Peer
	// Contains info about files in torrent. This can be nil at start for magnet downloads.
	info *metainfo.Info
	pieces []piece.Piece
	// Storage implementation to save the files in torrent.
	storage storage.Storage

	// A worker that opens and allocates files on the disk.
	allocator          *allocator.Allocator
	allocatorProgressC chan allocator.Progress
	allocatorResultC   chan *allocator.Allocator
	bytesAllocated     int64

	// A worker that does hash check of files on the disk.
	verifier          *verifier.Verifier
	verifierProgressC chan verifier.Progress
	verifierResultC   chan *verifier.Verifier
	checkedPieces     uint32

	// Active piece downloads are kept in this map.
	pieceDownloaders        map[*peer.Peer]*piecedownloader.PieceDownloader
	pieceDownloadersSnubbed map[*peer.Peer]*piecedownloader.PieceDownloader
	pieceDownloadersChoked  map[*peer.Peer]*piecedownloader.PieceDownloader

	// Active metadata downloads are kept in this map.
	infoDownloaders        map[*peer.Peer]*infodownloader.InfoDownloader
	infoDownloadersSnubbed map[*peer.Peer]*infodownloader.InfoDownloader

	ramNotifyC chan interface{}

	session *Session
	addedAt time.Time

	stopping 	bool
	stopC 		chan struct{}

	// Identifies the torrent being downloaded.
	infoHash [20]byte
	//info hash hex string
	id      string
	// Name of the torrent.
	name string


	// Bitfield for pieces we have. It is created after we got info.
	bitfield *bitfield.Bitfield

	// Protects bitfield writing from torrent loop and reading from announcer loop.
	mBitfield sync.RWMutex

	// Unique peer ID is generated per downloader.
	peerID [20]byte

	files  []allocator.File


	piecePicker *piecepicker.PiecePicker

	// We keep connected peers in this map after they complete handshake phase.
	peers map[*peer.Peer]struct{}

	// Keep recently seen peers to fill underpopulated PEX lists.
	recentlySeen pexlist.RecentlySeen

	// Unchoker implements an algorithm to select peers to unchoke based on their download speed.
	unchoker *unchoker.Unchoker

	pieceWriterResultC chan *piecewriter.PieceWriter

	// This channel is closed once all pieces are downloaded and verified.
	completeC chan struct{}

	// True after all pieces are download, verified and written to disk.
	completed bool

	// If any unrecoverable error occurs, it will be sent to this channel and download will be stopped.
	errC chan error

	// After listener has started, port will be sent to this channel.
	portC chan int

	// Contains the last error sent to errC.
	lastError error

	// When Stop() is called, it will close this channel to signal run() function to stop.
	closeC chan chan struct{}

	// Close() blocks until doneC is closed.
	doneC chan struct{}

	// These are the channels for sending a message to run() loop.
	//statsCommandC        chan statsRequest        // Stats()
	//trackersCommandC     chan trackersRequest     // Trackers()
	//peersCommandC        chan peersRequest        // Peers()
	//webseedsCommandC     chan webseedsRequest     // Webseeds()
	startCommandC        chan struct{}            // Start()
	stopCommandC         chan struct{}            // Stop()
	//announceCommandC     chan struct{}            // Announce()
	//verifyCommandC       chan struct{}            // Verify()
	//todo
	notifyErrorCommandC  chan notifyErrorCommand  // NotifyError()
	//notifyListenCommandC chan notifyListenCommand // NotifyListen()
	//addPeersCommandC     chan []*net.TCPAddr      // AddPeers()
	//addTrackersCommandC  chan []tracker.Tracker   // AddTrackers()

	// Keeps a list of peer addresses to connect.
	addrList *addrlist.AddrList

	// If not nil, torrent is announced to DHT periodically.
	dhtAnnouncer *announcer.DHTAnnouncer
	dhtPeersC    chan []*net.TCPAddr

	// When metadata of the torrent downloaded completely, a message is sent to this channel.
	infoDownloaderResultC chan *infodownloader.InfoDownloader

	// A ticker that ticks periodically to keep a certain number of peers unchoked.
	unchokeTicker *time.Ticker

	// Metrics
	downloadSpeed   metrics.Meter
	uploadSpeed     metrics.Meter
	bytesDownloaded metrics.Counter
	bytesUploaded   metrics.Counter
	bytesWasted     metrics.Counter
	seededFor       metrics.Counter

	seedDurationUpdatedAt time.Time
	seedDurationTicker    *time.Ticker

	// Peers that are sending corrupt data are banned.
	bannedPeerIPs map[string]struct{}

	// Piece buffers that are being downloaded are pooled to reduce load on GC.
	piecePool *bufferpool.Pool

	// Used to calculate canonical peer priority (BEP 40).
	// Initialized with value found in network interfaces.
	// Then, updated from "yourip" field in BEP 10 extension handshake message.
	externalIP net.IP

	// Set to true when manual verification is requested
	doVerify bool

	// If true, the torrent is stopped automatically when all pieces are downloaded.
	stopAfterDownload bool

	log logger.Logger
}

func newTorrent2(
	s *Session,
	addedAt time.Time,
	infoHash []byte,
	name string, // display name
	//fixedPeers []string,
	info *metainfo.Info,
	bf *bitfield.Bitfield,
	stats resumer.Stats, // initial stats from previous run
	stopAfterDownload bool,
	dataDir			  	string,
	id 					string,
) (*torrent, error) {
	if len(infoHash) != 20 {
		return nil, errors.New("invalid infoHash (must be 20 bytes)")
	}
	cfg := s.config
	var ih [20]byte
	copy(ih[:], infoHash)
	s.log.Debugln("new torrent!data dir:",dataDir)
	sto, err := filestorage.New(dataDir)
	if err != nil {
		return nil, err
	}

	t := &torrent{
		connectedPeers: 			make(map[p2pPeer.ID]*peer.Peer),
		incoimgPeersC: 				make(chan []p2pPeer.AddrInfo),
		incomingStreamC:			make(chan network.Stream),
		dhtNeedPeer:				false,
		session:                   s,
		addedAt:                   addedAt,
		infoHash:                  ih,
		id:						   id,
		name:                      name,
		storage:                   sto,
		info:                      info,
		bitfield:                  bf,
		dataDir: 				   dataDir,
		log:                       logger.New("torrent " + id),
		peerDisconnectedC:         make(chan *peer.Peer),
		messages:                  make(chan peer.Message),
		pieceMessagesC:            suspendchan.New(0),
		peers:                     make(map[*peer.Peer]struct{}),
		incomingPeers:             make(map[p2pPeer.ID]struct{}),
		outgoingPeers:             make(map[p2pPeer.ID]struct{}),
		pieceDownloaders:          make(map[*peer.Peer]*piecedownloader.PieceDownloader),
		pieceDownloadersSnubbed:   make(map[*peer.Peer]*piecedownloader.PieceDownloader),
		pieceDownloadersChoked:    make(map[*peer.Peer]*piecedownloader.PieceDownloader),
		peerSnubbedC:              make(chan *peer.Peer),
		infoDownloaders:           make(map[*peer.Peer]*infodownloader.InfoDownloader),
		infoDownloadersSnubbed:    make(map[*peer.Peer]*infodownloader.InfoDownloader),
		pieceWriterResultC:        make(chan *piecewriter.PieceWriter),
		completeC:                 make(chan struct{}),
		closeC:                    make(chan chan struct{}),
		startCommandC:             make(chan struct{}),
		stopCommandC:              make(chan struct{}),
		//announceCommandC:          make(chan struct{}),
		//verifyCommandC:            make(chan struct{}),
		//statsCommandC:             make(chan statsRequest),
		//trackersCommandC:          make(chan trackersRequest),
		//peersCommandC:             make(chan peersRequest),
		//webseedsCommandC:          make(chan webseedsRequest),
		notifyErrorCommandC:       make(chan notifyErrorCommand),
		//notifyListenCommandC:      make(chan notifyListenCommand),
		//addPeersCommandC:          make(chan []*net.TCPAddr),
		//addTrackersCommandC:       make(chan []tracker.Tracker),
		infoDownloaderResultC:     make(chan *infodownloader.InfoDownloader),
		allocatorProgressC:        make(chan allocator.Progress),
		allocatorResultC:          make(chan *allocator.Allocator),
		verifierProgressC:         make(chan verifier.Progress),
		verifierResultC:           make(chan *verifier.Verifier),
		bannedPeerIPs:             make(map[string]struct{}),
		dhtPeersC:                 make(chan []*net.TCPAddr, 1),
		externalIP:                externalip.FirstExternalIP(),
		downloadSpeed:             metrics.NilMeter{},
		uploadSpeed:               metrics.NilMeter{},
		bytesDownloaded:           metrics.NewCounter(),
		bytesUploaded:             metrics.NewCounter(),
		bytesWasted:               metrics.NewCounter(),
		seededFor:                 metrics.NewCounter(),
		ramNotifyC:                make(chan interface{}),
		doneC:                     make(chan struct{}),
		stopAfterDownload:         stopAfterDownload,

		stopC: 						make(chan struct{}),
	}
	t.bytesDownloaded.Inc(stats.BytesDownloaded)
	t.bytesUploaded.Inc(stats.BytesUploaded)
	t.bytesWasted.Inc(stats.BytesWasted)
	t.seededFor.Inc(stats.SeededFor)
	//var blocklistForOutgoingConns *blocklist.Blocklist
	//if cfg.BlocklistEnabledForOutgoingConnections {
	//	blocklistForOutgoingConns = s.blocklist
	//}
	t.addrList = addrlist.New(cfg.MaxPeerAddresses, &p2pPeer.AddrInfo{ID: s.host.ID(), Addrs: s.host.Addrs()})
	if t.info != nil {
		t.piecePool = bufferpool.New(int(t.info.PieceLength))
	}
	//n := t.copyPeerIDPrefix()
	//_, err := rand.Read(t.peerID[n:])
	//if err != nil {
	//	return nil, err
	//}
	//todo add peer id with user name

	t.unchoker = unchoker.New(cfg.UnchokedPeers, cfg.OptimisticUnchokedPeers)
	go t.run()
	return t, nil
}

func (t *torrent) getPeersForUnchoker() []unchoker.Peer {
	peers := make([]unchoker.Peer, 0, len(t.peers))
	for pe := range t.peers {
		peers = append(peers, pe)
	}
	return peers
}

func (t *torrent) Name() string {
	return t.name
}

func (t *torrent) InfoHash() []byte {
	b := make([]byte, 20)
	copy(b, t.infoHash[:])
	return b
}

// DisableLogging disables all log messages printed to console.
// This function needs to be called before creating a Session.
func DisableLogging() {
	logger.Disable()
}

func (t *torrent) AddIncomingStream(stream network.Stream)  {
	t.incomingStreamC <- stream
}

func (t *torrent) Close()  {
	t.close()
}
