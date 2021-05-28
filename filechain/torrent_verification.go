package filechain

import (
	"fmt"

	"github.com/fichain/go-file/external/peerprotocol"

	"github.com/fichain/go-file/internal/verifier"
)

func (t *torrent) handleVerifyCommand() {
	t.log.Info("verifying")
	t.doVerify = true
	if t.status() == Stopped {
		t.bitfield = nil
		t.start()
	} else {
		t.stop(nil)
	}
}

func (t *torrent) handleVerificationDone(ve *verifier.Verifier) {
	if t.verifier != ve {
		panic("invalid verifier")
	}
	t.verifier = nil

	if ve.Error != nil {
		t.stop(fmt.Errorf("file verification error: %s", ve.Error))
		return
	}

	// Now we have a constructed and verified bitfield.
	t.mBitfield.Lock()
	t.bitfield = ve.Bitfield
	t.mBitfield.Unlock()

	// Save the bitfield to resume db.
	//err := t.writeBitfield()
	//if err != nil {
	//	t.stop(err)
	//	return
	//}

	var haveMessages []peerprotocol.HaveMessage

	// Mark downloaded pieces.
	for i := uint32(0); i < t.bitfield.Len(); i++ {
		if t.bitfield.Test(i) {
			t.pieces[i].Done = true
			haveMessages = append(haveMessages, peerprotocol.HaveMessage{Index: i})
		} else {
			//t.log.Debugln("piece not done:", i)
		}
	}

	// We may detect missing pieces after verification. Then, status must be set from Seeding to Downloading.
	if !t.bitfield.All() {
		t.completed = false
		t.completeC = make(chan struct{})
	}

	if t.doVerify {
		// Stop after manual verification command.
		t.doVerify = false
		t.stop(nil)
		return
	}

	//Tell connected peers that pieces we have.
	for _, pe := range t.connectedPeers {
		for _, msg := range haveMessages {
			pe.SendMessage(msg)
		}
		t.updateInterestedState(pe)
	}

	t.log.Debugln("complete?", t.checkCompletion())
	if t.checkCompletion() && t.stopAfterDownload {
		t.log.Infoln("stop after download complete!")
		t.stop(nil)
		return
	}
	t.processQueuedMessages()
	//t.addFixedPeers()
	//t.startAcceptor()
	t.startAnnouncers()
	t.startPieceDownloaders()
}
