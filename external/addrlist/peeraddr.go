package addrlist

import (
	"sort"
	"time"

	"github.com/fichain/go-file/external/peerpriority"
	"github.com/fichain/go-file/external/peersource"
	"github.com/google/btree"
	"github.com/libp2p/go-libp2p-core/peer"
)

type peerAddr struct {
	pr		  *peer.AddrInfo
	//addr      *net.TCPAddr
	timestamp time.Time
	source    peersource.Source
	priority  peerpriority.Priority

	// index in AddrList.peerByTime slice
	index int
}

var _ btree.Item = (*peerAddr)(nil)

func (p *peerAddr) Less(than btree.Item) bool {
	return p.priority < than.(*peerAddr).priority
}

type byTimestamp []*peerAddr

var _ sort.Interface = (byTimestamp)(nil)

func (a byTimestamp) Len() int {
	return len(a)
}

func (a byTimestamp) Less(i, j int) bool {
	return a[i].timestamp.Before(a[j].timestamp)
}

func (a byTimestamp) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
	a[i].index = i
	a[j].index = j
}
