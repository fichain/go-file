package filechain

import (
	"github.com/fichain/go-file/internal/allocator"
	"github.com/fichain/go-file/internal/piecedownloader"
	"github.com/fichain/go-file/internal/verifier"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/rcrowley/go-metrics"

	"github.com/fichain/go-file/external/p2p"
	"github.com/fichain/go-file/external/peer"
	"github.com/fichain/go-file/external/peersource"
)

func (t *torrent) start() {
	// Do not start if already started.
	if t.errC != nil {
		return
	}

	// Stop announcing Stopped event if in "Stopping" state.
	if t.stoppedEventAnnouncer != nil {
		t.stoppedEventAnnouncer.Close()
		t.stoppedEventAnnouncer = nil
	}

	t.session.resumer.WriteStarted(t.id, true)

	t.log.Info("starting torrent")
	t.errC = make(chan error, 1)
	t.portC = make(chan int, 1)
	t.lastError = nil
	t.downloadSpeed = metrics.NewMeter()
	t.uploadSpeed = metrics.NewMeter()

	t.startStream()
	t.startAdvertise()
	if t.info != nil {
		if t.pieces != nil {
			if t.bitfield != nil {
				//t.addFixedPeers()
				t.startAnnouncers()
				t.startPieceDownloaders()
			} else {
				t.startVerifier()
			}
		} else {
			t.startAllocator()
		}
	} else {
		//t.addFixedPeers()
		//t.startStream()
		t.startAnnouncers()
		t.startInfoDownloaders()
	}
}

func (t *torrent) startStream()  {
	t.log.Debugln("startStream")
	t.session.host.SetStreamHandler(p2p.GenerateFileTransferProtocol(t.id), func(stream network.Stream) {
		t.incomingStreamC <- stream
	})

}

func (t *torrent) startInfoDownloaders() {
	t.log.Debugln("startInfoDownloaders")
	if t.info != nil {
		return
	}
	for len(t.infoDownloaders)-len(t.infoDownloadersSnubbed) < t.session.config.ParallelMetadataDownloads {
		id := t.nextInfoDownload()
		if id == nil {
			break
		}
		pe := id.Peer.(*peer.Peer)
		t.infoDownloaders[pe] = id
		id.RequestBlocks(t.maxAllowedRequests(pe))
		id.Peer.(*peer.Peer).ResetSnubTimer()
	}
}

func (t *torrent) maxAllowedRequests(pe *peer.Peer) int {
	ret := t.session.config.DefaultRequestsOut
	if pe.ExtensionHandshake != nil && pe.ExtensionHandshake.RequestQueue > 0 {
		ret = pe.ExtensionHandshake.RequestQueue
	}
	if ret > t.session.config.MaxRequestsOut {
		ret = t.session.config.MaxRequestsOut
	}
	return ret
}

func (t *torrent) startVerifier() {
	t.log.Debugln("startVerifier")
	if t.verifier != nil {
		panic("verifier exists")
	}
	if len(t.pieces) == 0 {
		panic("zero length pieces")
	}
	t.verifier = verifier.New()
	go t.verifier.Run(t.pieces, t.verifierProgressC, t.verifierResultC)
}

func (t *torrent) startAllocator() {
	t.log.Debugln("startAllocator")
	if t.allocator != nil {
		panic("allocator exists")
	}
	t.allocator = allocator.New()
	go t.allocator.Run(t.info, t.storage, t.allocatorProgressC, t.allocatorResultC)
}
//
//func (t *torrent) addFixedPeers() {
//	for _, pe := range t.fixedPeers {
//		_ = t.addPeerString(pe)
//	}
//}
//

func (t *torrent) startAdvertise()  {
	t.log.Debugln("startAdvertise")
	err := p2p.Advertise(t.session.routeDiscovery, t.id)
	if err != nil {
		t.log.Errorln("advertise error!", err)
	} else {
		t.log.Infoln("advertise success")
	}
}

func (t *torrent) startAnnouncers() {
	t.log.Debugln("startAnnouncers")
	id := t.id
	peerAddrs, err := p2p.FindProviders(t.session.routeDiscovery, id, 0)
	if err != nil {
		t.log.Errorln("find peers error!", err)
	} else {
		if len(peerAddrs) != 0 {
			t.handleNewPeers(peerAddrs, peersource.DHT)
		}
	}
	//t.session.routeDiscovery.Advertise(context.Background(), hex.EncodeToString(t.infoHash[:]))
}

func (t *torrent) startPieceDownloaders() {
	if t.status() != Downloading {
		return
	}
	//t.log.Debugln("start piece download!")
	for _, pe := range t.connectedPeers {
		if !pe.Downloading {
			t.startPieceDownloaderFor(pe)
		}
	}
}

func (t *torrent) startPieceDownloaderFor(pe *peer.Peer) {
	if t.status() != Downloading {
		return
	}
	//t.log.Debugln("start piece download, peer is:", pe.ID)
	if t.session.ram == nil {
		t.startSinglePieceDownloader(pe)
		return
	}
	ok := t.session.ram.Request(t.id, pe, int64(t.info.PieceLength), t.ramNotifyC, pe.Done())
	if ok {
		t.startSinglePieceDownloader(pe)
	}
}

func (t *torrent) startSinglePieceDownloader(pe *peer.Peer) {
	var started bool
	defer func() {
		if !started && t.session.ram != nil {
			t.session.ram.Release(int64(t.info.PieceLength))
		}
	}()
	if t.status() != Downloading {
		return
	}
	//t.log.Debugln("start single piece download, peer is:", pe.ID)
	pi, allowedFast := t.piecePicker.PickFor(pe)
	if pi == nil {
		t.log.Debugln("pi nil", pe.ID)
		return
	}
	pd := piecedownloader.New(pi, pe, allowedFast, t.piecePool.Get(int(pi.Length)))
	if _, ok := t.pieceDownloaders[pe]; ok {
		panic("peer already has a piece downloader")
	}
	t.log.Debugf("requesting piece #%d from peer %s", pi.Index, pe.Stream.Conn().RemotePeer().Pretty())
	t.pieceDownloaders[pe] = pd
	pe.Downloading = true
	pd.RequestBlocks(t.maxAllowedRequests(pe))
	pe.ResetSnubTimer()
	started = true
}
