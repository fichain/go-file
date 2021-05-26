package filechain

import (
	p2p2 "github.com/fichain/go-file/external/p2p"
	"github.com/libp2p/go-libp2p-core/network"
)

func (s *Session)handleStream()  {
	s.host.SetStreamHandler(p2p2.FileTransferProtocol, func(stream network.Stream) {
		//first hand shake
		flag := false
		s.mTorrents.RLock()
		for _,v := range s.torrents {
			if stream.Protocol() != p2p2.GenerateFileTransferProtocol(v.id) {
				continue
			}
			flag = true
			v.AddIncomingStream(stream)
			break
		}
		s.mTorrents.RUnlock()
		if !flag{
			s.log.Errorln("mistake income connection!", stream.Conn().RemotePeer().String())
			stream.Close()
		}
	})
}