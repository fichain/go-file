package p2p

import "github.com/libp2p/go-libp2p-core/protocol"

const (
	FileTransferProtocol = "/filchain/transfer/1.0.0"
)

func GenerateFileTransferProtocol(infohash string) protocol.ID {
	return protocol.ID(FileTransferProtocol + "/" + infohash)
}