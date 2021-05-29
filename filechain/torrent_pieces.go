package filechain

import (
	"fmt"
)

func (t *torrent) writeBitfield() error {
	err := t.session.resumer.WriteBitfield(t.id, t.bitfield.Bytes())
	if err != nil {
		err = fmt.Errorf("cannot write bitfield to resume db: %s", err)
		t.log.Errorln(err)
	}
	return err
}

func (t *torrent) checkCompletion() bool {
	if t.completed {
		return true
	}
	if !t.bitfield.All() {
		return false
	}
	t.completed = true
	close(t.completeC)
	//todo
	for _, pe := range t.connectedPeers{
		if !pe.PeerInterested {
			t.closePeer(pe)
		}
	}
	t.addrList.Reset()
	for _, pd := range t.pieceDownloaders {
		t.closePieceDownloader(pd)
		pd.CancelPending()
	}
	t.piecePicker = nil
	//t.updateSeedDuration(time.Now())
	return true
}
