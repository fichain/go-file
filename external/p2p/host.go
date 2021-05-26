package p2p

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"

	logging "github.com/ipfs/go-log"
	connmgr "github.com/libp2p/go-libp2p-connmgr"
	noise "github.com/libp2p/go-libp2p-noise"
	libp2ptls "github.com/libp2p/go-libp2p-tls"

	ma "github.com/multiformats/go-multiaddr"
)

var logger = logging.Logger("p2p")

func convertPeers(peers []string) []peer.AddrInfo {
	pinfos := make([]peer.AddrInfo, len(peers))
	for i, addr := range peers {
		maddr := ma.StringCast(addr)
		p, err := peer.AddrInfoFromP2pAddr(maddr)
		if err != nil {
			log.Fatalln(err)
		}
		pinfos[i] = *p
	}
	return pinfos
}

func NewRoutedHost(listenPort int, bootstrapPeers []string, priv crypto.PrivKey) (host.Host, error) {
	bpeers := convertPeers(bootstrapPeers)

	ctx := context.Background()

	opts := []libp2p.Option{
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", listenPort)),
		libp2p.Identity(priv),
		libp2p.DefaultTransports,
		libp2p.DefaultMuxers,
		libp2p.Security(libp2ptls.ID, libp2ptls.New),
		libp2p.Security(noise.ID, noise.New),
		//libp2p.NATPortMap(),
		//libp2p.EnableAutoRelay(),
		//libp2p.EnableNATService(),
		libp2p.ConnectionManager(connmgr.NewConnManager(
			100,         // Lowwater
			400,         // HighWater,
			time.Minute, // GracePeriod
		)),
	}
	basicHost, err := libp2p.New(ctx, opts...)
	if err != nil {
		panic(err)
	}
	logger.Debug("")

	var wg sync.WaitGroup
	for _, peerAddr := range bpeers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := basicHost.Connect(ctx, peerAddr); err != nil {
				logger.Warn(err)
			} else {
				logger.Info("Connection established with bootstrap node:", peerAddr.String())
			}
		}()
	}
	wg.Wait()

	logger.Infof("Hello World, my hosts ID is %s/%s\n", basicHost.Addrs()[0].String(), basicHost.ID())
	return basicHost, nil
}