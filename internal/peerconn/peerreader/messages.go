package peerreader

import (
	"github.com/fichain/go-file/internal/bufferpool"
	"github.com/fichain/go-file/internal/peerprotocol"
)

// Piece message that is read from peers.
// Data of the piece is wrapped with a bufferpool.Buffer object.
type Piece struct {
	peerprotocol.PieceMessage
	Buffer bufferpool.Buffer
}
