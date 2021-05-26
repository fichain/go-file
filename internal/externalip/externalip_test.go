package externalip

import (
	"fmt"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"
	"testing"
)

func StringsToAddrs(addrStrings []string) (maddrs []multiaddr.Multiaddr, err error) {
	for _, addrString := range addrStrings {
		addr, err := multiaddr.NewMultiaddr(addrString)
		if err != nil {
			return maddrs, err
		}
		maddrs = append(maddrs, addr)
	}
	return
}

func addrmultiaddrtoaddr(maddrs []multiaddr.Multiaddr) *peer.AddrInfo  {
	if len(maddrs) == 0 {
		return nil
	}

	_, id := peer.SplitAddr(maddrs[0])
	return &peer.AddrInfo{
		ID: id,
		Addrs: maddrs,
	}
}

func TestGetExternalTcp(t *testing.T) {
	//case 1 have public ip
	addrl1 := []string{
		"/ip4/192.168.2.244/tcp/59716",
		"/ip4/127.0.0.1/tcp/59716",
		"/ip6/::1/tcp/59717",
		"/ip4/198.168.2.244/tcp/59716",
	}
	maddrs1, err := StringsToAddrs(addrl1)
	if err != nil {
		t.Errorf("err happend! %s", err)
		t.FailNow()
	}
	addr1 := addrmultiaddrtoaddr(maddrs1)
	if addr1 == nil {
		t.Error("addr1 error!!")
		t.FailNow()
	}

	netTcp1 := GetExternalFirstTcp(addr1)
	if netTcp1 == nil {
		t.Error("addr1 error!!")
		t.FailNow()
	}
	fmt.Println(netTcp1.String())

	if netTcp1.String() != "198.168.2.244:59716" {
		t.Error("tcp addr1 error!!")
		t.FailNow()
	}

	// case 2 no public ip
	addrl2 := []string{
		"/ip4/192.168.2.244/tcp/59716",
		"/ip4/127.0.0.1/tcp/59716",
		"/ip6/::1/tcp/59717",
	}
	maddrs2, err := StringsToAddrs(addrl2)
	if err != nil {
		t.Errorf("err happend2! %s", err)
		t.FailNow()
	}
	addr2 := addrmultiaddrtoaddr(maddrs2)
	if addr2 == nil {
		t.Error("addr2 error!!")
		t.FailNow()
	}

	netTcp2 := GetExternalFirstTcp(addr2)

	if netTcp2 != nil {
		t.Error("tcp addr2 error!!")
		t.FailNow()
	}

}
