package filechain

import (
	"github.com/fichain/go-file/internal/handshaker/incominghandshaker"
	"github.com/fichain/go-file/internal/handshaker/outgoinghandshaker"
	//"github.com/fichain/go-file/internal/handshaker/incominghandshaker"
	//"github.com/fichain/go-file/internal/handshaker/outgoinghandshaker"
	//"github.com/fichain/go-file/internal/tracker"
	"github.com/rcrowley/go-metrics"
)

func (t *torrent) handleStopped() {
	t.stoppedEventAnnouncer = nil
	t.errC <- t.lastError
	t.errC = nil
	t.portC = nil
	if t.doVerify {
		t.bitfield = nil
		t.start()
	} else {
		t.log.Info("torrent has stopped")
	}
}

func (t *torrent) stop(err error) {
	s := t.status()
	if s == Stopping || s == Stopped {
		return
	}

	t.log.Info("stopping torrent")
	t.lastError = err
	if err != nil && err != errClosed {
		t.log.Error(err)
	}

	t.stopAcceptor()
	t.stopPeers()
	t.stopPiecedownloaders()
	t.stopInfoDownloaders()

	if t.bitfield != nil {
		_ = t.writeBitfield()
	}

	// Closing data is necessary to cancel ongoing IO operations on files.
	t.closeData()
	// Data must be closed before closing Allocator.
	t.stopAllocator()
	// Data must be closed before closing Verifier.
	t.stopVerifier()

	t.stopOutgoingHandshakers()
	t.stopIncomingHandshakers()

	t.resetSpeeds()

	//todo dht

	t.addrList.Reset()
}

func (t *torrent) stopAllocator() {
	t.log.Debugln("stopping allocator")
	if t.allocator != nil {
		t.allocator.Close()
		t.allocator = nil
	}
}

func (t *torrent) stopVerifier() {
	t.log.Debugln("stopping verifier")
	if t.verifier != nil {
		t.verifier.Close()
		t.verifier = nil
	}
}

func (t *torrent) stopWebseedDownloads() {
	//for _, src := range t.webseedSources {
	//	t.closeWebseedDownloader(src)
	//}
}

func (t *torrent) resetSpeeds() {
	t.downloadSpeed.Stop()
	t.downloadSpeed = metrics.NilMeter{}
	t.uploadSpeed.Stop()
	t.uploadSpeed = metrics.NilMeter{}
}

//todo
func (t *torrent) stopOutgoingHandshakers() {
	t.log.Debugln("stopping outgoing handshakers")
	for oh := range t.outgoingHandshakers {
		oh.Close()
	}
	t.outgoingHandshakers = make(map[*outgoinghandshaker.OutgoingHandshaker]struct{})
}

//todo
func (t *torrent) stopIncomingHandshakers() {
	t.log.Debugln("stopping incoming handshakers")
	for ih := range t.incomingHandshakers {
		ih.Close()
	}
	t.incomingHandshakers = make(map[*incominghandshaker.IncomingHandshaker]struct{})
}

func (t *torrent) closeData() {
	t.log.Debugln("closing open files")
	for _, f := range t.files {
		err := f.Storage.Close()
		if err != nil {
			t.log.Error(err)
		}
	}
	t.files = nil
	t.pieces = nil
	t.piecePicker = nil
	t.bytesAllocated = 0
	t.checkedPieces = 0
}

func (t *torrent) stopPeriodicalAnnouncers() {
	t.log.Debugln("stopping announcers")
	for _, an := range t.announcers {
		an.Close()
	}
	t.announcers = nil
	if t.dhtAnnouncer != nil {
		t.dhtAnnouncer.Close()
		t.dhtAnnouncer = nil
	}
}

func (t *torrent) stopAcceptor() {
	t.log.Debugln("stopping acceptor")
	if t.acceptor != nil {
		t.acceptor.Close()
	}
	t.acceptor = nil
}

func (t *torrent) stopPeers() {
	t.log.Debugln("closing peer connections")
	for _, p := range t.connectedPeers {
		t.closePeer(p)
	}
}

func (t *torrent) stopInfoDownloaders() {
	t.log.Debugln("stopping info downloaders")
	for _, id := range t.infoDownloaders {
		t.closeInfoDownloader(id)
	}
}

func (t *torrent) stopPiecedownloaders() {
	t.log.Debugln("stopping piece downloaders")
	for _, pd := range t.pieceDownloaders {
		t.closePieceDownloader(pd)
	}
}
