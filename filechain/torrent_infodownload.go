package filechain

import (
	"github.com/fichain/go-file/external/peerprotocol"
	"github.com/fichain/go-file/internal/infodownloader"
)

func (t *torrent) nextInfoDownload() *infodownloader.InfoDownloader {
	for _, pe := range t.connectedPeers {
		if _, ok := t.infoDownloaders[pe]; ok {
			continue
		}
		if pe.ExtensionHandshake == nil {
			continue
		}
		if pe.ExtensionHandshake.MetadataSize == 0 {
			continue
		}
		//if pe.ExtensionHandshake.MetadataSize > int(t.session.config.MaxMetadataSize) {
		//	t.log.Debugf("metadata size larger than allowed: %d", pe.ExtensionHandshake.MetadataSize)
		//	continue
		//}
		_, ok := pe.ExtensionHandshake.M[peerprotocol.ExtensionKeyMetadata]
		if !ok {
			continue
		}
		t.log.Debugln("downloading info from", pe.ID)
		return infodownloader.New(pe)
	}
	return nil
}
