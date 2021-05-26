package externalip

import (
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"
	"net"
	"strconv"

	"github.com/cenkalti/log"
)

var ips []net.IP

func init() {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		log.Warningln("cannot get interface addresses:", err)
		return
	}
	for _, addr := range addrs {
		in, ok := addr.(*net.IPNet)
		if !ok {
			continue
		}
		i4 := in.IP.To4()
		if i4 == nil {
			continue
		}
		if !isPublicIP(i4) {
			continue
		}
		ips = append(ips, i4)
	}
}

func isPublicIP(ip4 net.IP) bool {
	if ip4.IsLoopback() || ip4.IsLinkLocalMulticast() || ip4.IsLinkLocalUnicast() {
		return false
	}
	switch {
	case ip4[0] == 10:
		return false
	case ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31:
		return false
	case ip4[0] == 192 && ip4[1] == 168:
		return false
	default:
		return true
	}
}

// IsExternal returns true if the given IP matches one of the IP address of the external network interfaces on the server.
func IsExternal(ip net.IP) bool {
	for i := range ips {
		if ip.Equal(ips[i]) {
			return true
		}
	}
	return false
}

// FirstExternalIP returns the first external IP of the network interfaces on the server.
func FirstExternalIP() net.IP {
	if len(ips) == 0 {
		return nil
	}
	return ips[0]
}

func GetExternalFirstTcp(addr *peer.AddrInfo) *net.TCPAddr  {
	var (
		netip net.IP
		found bool
		port int
	)
	for _, ad := range addr.Addrs{
		tmp, err := ad.ValueForProtocol(multiaddr.P_IP4)
		if err != nil {
			continue
		}
		netip = net.ParseIP(tmp)

		if !isPublicIP(netip.To4()){
			continue
		}

		tmp2, err := ad.ValueForProtocol(multiaddr.P_TCP)
		if err != nil {
			continue
		}
		port, err = strconv.Atoi(tmp2)

		if err != nil {
			continue
		}

		found = true
		break
	}

	if !found {
		return nil
	}

	return &net.TCPAddr{
		IP: netip,
		Port: port,
	}
}

func GetInternalFirstTcp(addr *peer.AddrInfo) *net.TCPAddr  {
	var (
		netip net.IP
		found bool
		port int
	)
	for _, ad := range addr.Addrs{
		tmp, err := ad.ValueForProtocol(multiaddr.P_IP4)
		if err != nil {
			continue
		}
		netip = net.ParseIP(tmp)

		//if !isPublicIP(netip.To4()){
		//	continue
		//}

		tmp2, err := ad.ValueForProtocol(multiaddr.P_TCP)
		if err != nil {
			continue
		}
		port, err = strconv.Atoi(tmp2)

		if err != nil {
			continue
		}

		found = true
		break
	}

	if !found {
		return nil
	}

	return &net.TCPAddr{
		IP: netip,
		Port: port,
	}
}