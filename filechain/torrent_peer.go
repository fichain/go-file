package filechain

import (
	"context"
	"github.com/fichain/go-file/external/p2p"
	"github.com/fichain/go-file/external/peer"
	"github.com/fichain/go-file/external/peerprotocol"
	"github.com/fichain/go-file/external/peersource"

	"github.com/fichain/go-file/internal/bitfield"
	p2pPeer "github.com/libp2p/go-libp2p-core/peer"

	"github.com/libp2p/go-libp2p-core/network"
)

func (t *torrent) setNeedMorePeers(val bool) {
	t.mDhtNeedPeer.Lock()
	t.dhtNeedPeer = val
	t.mDhtNeedPeer.Unlock()
}

func (t *torrent)handleNewIncomingStream(stream network.Stream)  {
	id := stream.Conn().RemotePeer()
	if _, ok := t.connectedPeers[id]; ok {
		t.log.Infof("duplicate peer, id: %s, close stream!", stream.Conn().RemotePeer().Pretty())
		stream.Close()
		return
	}

	pe := peer.New(t.session.host, stream, peersource.Incoming, t.infoHash, t.session.config.PieceReadTimeout, t.session.config.RequestTimeout, t.session.config.MaxRequestsIn, t.session.bucketDownload, t.session.bucketUpload)
	t.connectedPeers[stream.Conn().RemotePeer()] = pe
	//	go pe.Run(t.messages, t.pieceMessagesC.SendC(), t.peerSnubbedC, t.peerDisconnectedC)
	t.startPeer(pe)
	//go pe.Run(t.messages, t.pieceMessagesC.SendC(), t.peerSnubbedC, t.peerDisconnectedC)
}

func (t *torrent) handleNewPeers(addrs []p2pPeer.AddrInfo, source peersource.Source) {
	t.log.Debugf("received %d peers from %s\n", len(addrs), source)
	t.setNeedMorePeers(false)
	if status := t.status(); status == Stopped || status == Stopping {
		return
	}
	if !t.completed {
		//addrs = t.filterBannedIPs(addrs)
		t.log.Debugln("not complete push addr to list")
		t.addrList.Push(t.addrList.ToAddrs(addrs), source)
		t.dialAddresses()
	}
}

func (t *torrent) dialAddresses() {
	if t.completed {
		return
	}


	peersConnected := func() int {
		return len(t.connectedPeers)
	}

	for peersConnected() < t.session.config.MaxPeerDial {
		addr, src := t.addrList.Pop()
		if addr == nil {
			t.setNeedMorePeers(true)
			return
		}
		t.log.Debugln("pop addr:", addr.String(), t.addrList.Len())

		if _, ok := t.connectedPeers[addr.ID]; ok {
			continue
		}

		s, err := t.session.host.NewStream(context.Background(), addr.ID, p2p.GenerateFileTransferProtocol(t.id))
		if err != nil {
			t.log.Errorln("create new stream error:", err)
			continue
		}
		t.log.Debugln("create new stream success!", s.Conn().RemotePeer().Pretty())
		pe := peer.New(t.session.host, s, src, t.infoHash, t.session.config.PieceReadTimeout, t.session.config.RequestTimeout, t.session.config.MaxRequestsIn, t.session.bucketDownload, t.session.bucketUpload)
		t.connectedPeers[addr.ID] = pe
		t.startPeer(pe)
	}
}

func (t *torrent)startPeer(pe *peer.Peer)  {
	t.log.Debugln("start peer!", pe.Stream.Conn().RemotePeer().Pretty())
	if t.info != nil {
		pe.Bitfield = bitfield.New(t.info.NumPieces)
	}
	t.log.Debugln("run peer!", pe.Stream.Conn().RemotePeer().Pretty())
	go pe.Run(t.messages, t.pieceMessagesC.SendC(), t.peerSnubbedC, t.peerDisconnectedC)
	//todo metrcis
	//t.session.metrics.Peers.Inc(1)
	t.log.Debugln("send first message", pe.Stream.Conn().RemotePeer().Pretty())
	t.sendFirstMessage(pe)
	//t.recentlySeen.Add(pe.Addr())
}

func (t *torrent) sendFirstMessage(p *peer.Peer) {
	bf := t.bitfield
	switch {
	case p.FastEnabled && bf != nil && bf.All():
		//fmt.Println("send have all message!")
		msg := peerprotocol.HaveAllMessage{}
		p.SendMessage(msg)
	case p.FastEnabled && (bf == nil || bf.Count() == 0):
		msg := peerprotocol.HaveNoneMessage{}
		p.SendMessage(msg)
	case bf != nil:
		bitfieldData := make([]byte, len(bf.Bytes()))
		copy(bitfieldData, bf.Bytes())
		msg := peerprotocol.BitfieldMessage{Data: bitfieldData}
		p.SendMessage(&msg)
	}

	var metadataSize uint32
	if t.info != nil {
		metadataSize = uint32(len(t.info.Bytes))
	}

	//send
	extHandshakeMsg := peerprotocol.NewExtensionHandshake(metadataSize, "1.0", t.externalIP, t.session.config.MaxRequestsIn)
	msg := peerprotocol.ExtensionMessage{
		ExtendedMessageID: peerprotocol.ExtensionIDHandshake,
		Payload:           extHandshakeMsg,
	}
	//t.log.Debugf("first hand shake msg: %v", extHandshakeMsg)
	p.SendMessage(msg)
}

// Process messages received while we don't have metadata yet.
func (t *torrent) processQueuedMessages() {
	for _, pe := range t.connectedPeers {
		for _, msg := range pe.Messages {
			pm := peer.Message{Peer: pe, Message: msg}
			t.handlePeerMessage(pm)
		}
		pe.Messages = nil
	}
}