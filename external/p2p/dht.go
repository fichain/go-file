package p2p

import (
	"context"
	"github.com/libp2p/go-libp2p-core/peer"
	"time"

	coreDiscovery "github.com/libp2p/go-libp2p-core/discovery"
	"github.com/libp2p/go-libp2p-core/host"
	discovery "github.com/libp2p/go-libp2p-discovery"
	dht "github.com/libp2p/go-libp2p-kad-dht"
)

//todo mode server
func NewRoutedDiscovery(h host.Host) (*discovery.RoutingDiscovery, error) {
	ctx := context.Background()
	dht, err := dht.New(ctx, h, dht.Mode(dht.ModeServer))
	if err != nil {
		return nil, err
	}

	//dht.FindProvidersAsync()
	routingDiscovery := discovery.NewRoutingDiscovery(dht)

	dht.RefreshRoutingTable()

	return routingDiscovery, nil
}

//todo is thread safe?
func FindProviders(discovery *discovery.RoutingDiscovery, id string, limit int) ([]peer.AddrInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second * 60)
	defer cancel()

	peerAddrC, err := discovery.FindPeers(ctx, id, coreDiscovery.Limit(limit))
	if err != nil {
		return nil, err
	}

	var addrs []peer.AddrInfo
	for p := range peerAddrC {
		addrs = append(addrs, p)
	}
	return addrs, err
}

func Advertise(discovery *discovery.RoutingDiscovery, id string) error {
	_, err := discovery.Advertise(context.Background(), id)
	return err
}