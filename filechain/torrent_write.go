package filechain

import (
	"fmt"

	"github.com/fichain/go-file/external/peer"
	"github.com/fichain/go-file/external/peerprotocol"

	"github.com/fichain/go-file/internal/piecewriter"
)

func (t *torrent) handlePieceWriteDone(pw *piecewriter.PieceWriter) {
	pw.Piece.Writing = false

	t.pieceMessagesC.Resume()

	pw.Buffer.Release()

	if !pw.HashOK {
		t.bytesWasted.Inc(int64(len(pw.Buffer.Data)))
		switch src := pw.Source.(type) {
		case *peer.Peer:
			t.log.Debugln("received corrupt piece from peer", src.String())
			t.closePeer(src)
			//todo banned peer
			//t.bannedPeerIPs[src.IP()] = struct{}{}
		default:
			panic("unhandled piece source")
		}
		t.startPieceDownloaders()
		return
	}
	if pw.Error != nil {
		t.stop(pw.Error)
		return
	}

	pw.Piece.Done = true
	if t.bitfield.Test(pw.Piece.Index) {
		panic(fmt.Sprintf("already have the piece #%d", pw.Piece.Index))
	}
	t.mBitfield.Lock()
	t.bitfield.Set(pw.Piece.Index)
	t.mBitfield.Unlock()

	if t.piecePicker != nil {
		for _, pe := range t.piecePicker.RequestedPeers(pw.Piece.Index) {
			pd2 := t.pieceDownloaders[pe]
			t.closePieceDownloader(pd2)
			pd2.CancelPending()
			t.startPieceDownloaderFor(pe)
		}
	}

	// Tell everyone that we have this piece
	for pe := range t.peers {
		t.updateInterestedState(pe)
		if pe.Bitfield.Test(pw.Piece.Index) {
			// Skip peers having the piece to save bandwidth
			continue
		}
		msg := peerprotocol.HaveMessage{Index: pw.Piece.Index}
		pe.SendMessage(msg)
	}

	completed := t.checkCompletion()
	if completed {
		t.log.Info("download completed")
		err := t.writeBitfield()
		if err != nil {
			t.stop(err)
		} else if t.stopAfterDownload {
			t.stop(nil)
		}
	}
}
