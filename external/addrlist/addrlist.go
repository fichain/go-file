package addrlist

import (
	"fmt"
	"github.com/fichain/go-file/internal/externalip"
	"github.com/multiformats/go-multiaddr"
	"net"
	"sort"
	"time"

	//"github.com/fichain/go-file/internal/blocklist"
	"github.com/fichain/go-file/external/peerpriority"
	"github.com/fichain/go-file/external/peersource"
	"github.com/google/btree"
	"github.com/libp2p/go-libp2p-core/peer"
)

// AddrList contains peer addresses that are ready to be connected.
type AddrList struct {
	peerByTime     []*peerAddr
	peerByPriority *btree.BTree

	maxItems   int
	//listenPort int

	clientPeer	*peer.AddrInfo
	clientTcp	*net.TCPAddr
	//blocklist  *blocklist.Blocklist
	//todo
	filter 	*multiaddr.Filters
	countBySource map[peersource.Source]int
}

// New returns a new AddrList.
func New(maxItems int, clientPeer *peer.AddrInfo) *AddrList {
	netTcp := externalip.GetExternalFirstTcp(clientPeer)
	if netTcp == nil {
		netTcp = externalip.GetInternalFirstTcp(clientPeer)
	}

	return &AddrList{
		peerByPriority: btree.New(2),

		maxItems:      maxItems,
		//listenPort:    listenPort,
		clientPeer:     clientPeer,
		clientTcp: 		netTcp,
		//blocklist:     blocklist,
		countBySource: make(map[peersource.Source]int),
	}
}

// Reset empties the address list.
func (d *AddrList) Reset() {
	d.peerByTime = nil
	d.peerByPriority.Clear(false)
	d.countBySource = make(map[peersource.Source]int)
}

// Len returns the number of addresses in the list.
func (d *AddrList) Len() int {
	return d.peerByPriority.Len()
}

// LenSource returns the number of addresses for a single source.
func (d *AddrList) LenSource(s peersource.Source) int {
	return d.countBySource[s]
}

// Pop returns the next address. The returned address is removed from the list.
func (d *AddrList) Pop() (*peer.AddrInfo, peersource.Source) {
	item := d.peerByPriority.DeleteMax()
	if item == nil {
		return nil, 0
	}
	p := item.(*peerAddr)
	d.peerByTime[p.index] = nil
	d.countBySource[p.source]--
	return p.pr, p.source
}

// Push adds a new address to the list. Does nothing if the address is already in the list.
func (d *AddrList) Push(addrs []*peer.AddrInfo, source peersource.Source) {
	now := time.Now()
	var added int
	for _, ad := range addrs {
		fmt.Println("push addr:", ad.String())
		if len(ad.Addrs) == 0{
			continue
		}
		if d.clientPeer.ID == ad.ID {
			continue
		}

		blockFlag := false
		for _, maad := range ad.Addrs {
			if d.filter != nil && d.filter.AddrBlocked(maad) {
				blockFlag = true
				break
			}
		}
		if blockFlag {
			continue
		}

		netTcp := externalip.GetExternalFirstTcp(ad)
		if netTcp == nil {
			//continue
			netTcp = externalip.GetInternalFirstTcp(ad)
		}

		//fmt.Println("tcp:", )
		p := &peerAddr{
			pr:      ad,
			timestamp: now,
			source:    source,
			priority:  peerpriority.Calculate(netTcp, d.clientTcp),
		}
		item := d.peerByPriority.ReplaceOrInsert(p)
		fmt.Println("push res:", item, d.Len())
		if item != nil {
			prev := item.(*peerAddr)
			d.peerByTime[prev.index] = p
			p.index = prev.index
			d.countBySource[prev.source]--
		} else {
			d.peerByTime = append(d.peerByTime, p)
			p.index = len(d.peerByTime) - 1
		}
		added++
	}
	d.filterNils()
	sort.Sort(byTimestamp(d.peerByTime))
	d.countBySource[source] += added

	delta := d.peerByPriority.Len() - d.maxItems
	if delta > 0 {
		d.removeExcessItems(delta)
		d.filterNils()
		d.countBySource[source] -= delta
	}
	if len(d.peerByTime) != d.peerByPriority.Len() {
		panic("addr list data structures not in sync")
	}
}

func (d *AddrList) filterNils() {
	b := d.peerByTime[:0]
	for _, x := range d.peerByTime {
		if x != nil {
			b = append(b, x)
			x.index = len(b) - 1
		}
	}
	d.peerByTime = b
}

func (d *AddrList) removeExcessItems(delta int) {
	for i := 0; i < delta; i++ {
		d.peerByPriority.Delete(d.peerByTime[i])
		d.peerByTime[i] = nil
	}
}

func (d *AddrList) ToAddrs(addrs []peer.AddrInfo) []*peer.AddrInfo {
	var res []*peer.AddrInfo
	for _,v := range addrs{
		res = append(res, &v)
	}
	return res
}
//func (d *AddrList) clientAddr() *net.TCPAddr {
//	if d.clientTcp == nil {
//		return &net.TCPAddr{
//			IP:   net.IPv4(0, 0, 0, 0),
//			Port: d.listenPort,
//		}
//	} else {
//		return d.clientTcp
//	}
//}
