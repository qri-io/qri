package manet

import (
	"fmt"
	"net"
	"sync"

	ma "gx/ipfs/QmSWLfmj5frN9xVLMMN846dMDriy5wN5jeghUm7aTW3DAG/go-multiaddr"
)

type FromNetAddrFunc func(a net.Addr) (ma.Multiaddr, error)
type ToNetAddrFunc func(ma ma.Multiaddr) (net.Addr, error)

var defaultCodecs *CodecMap

func init() {
	defaultCodecs = NewCodecMap()
	defaultCodecs.RegisterNetCodec(tcpAddrSpec)
	defaultCodecs.RegisterNetCodec(udpAddrSpec)
	defaultCodecs.RegisterNetCodec(ip4AddrSpec)
	defaultCodecs.RegisterNetCodec(ip6AddrSpec)
	defaultCodecs.RegisterNetCodec(ipnetAddrSpec)
}

type CodecMap struct {
	codecs       map[string]*NetCodec
	addrParsers  map[string]FromNetAddrFunc
	maddrParsers map[string]ToNetAddrFunc
	lk           sync.Mutex
}

func NewCodecMap() *CodecMap {
	return &CodecMap{
		codecs:       make(map[string]*NetCodec),
		addrParsers:  make(map[string]FromNetAddrFunc),
		maddrParsers: make(map[string]ToNetAddrFunc),
	}
}

type NetCodec struct {
	// NetAddrNetworks is an array of strings that may be returned
	// by net.Addr.Network() calls on addresses belonging to this type
	NetAddrNetworks []string

	// ProtocolName is the string value for Multiaddr address keys
	ProtocolName string

	// ParseNetAddr parses a net.Addr belonging to this type into a multiaddr
	ParseNetAddr FromNetAddrFunc

	// ConvertMultiaddr converts a multiaddr of this type back into a net.Addr
	ConvertMultiaddr ToNetAddrFunc

	// Protocol returns the multiaddr protocol struct for this type
	Protocol ma.Protocol
}

func RegisterNetCodec(a *NetCodec) {
	defaultCodecs.RegisterNetCodec(a)
}

func (cm *CodecMap) RegisterNetCodec(a *NetCodec) {
	cm.lk.Lock()
	defer cm.lk.Unlock()
	cm.codecs[a.ProtocolName] = a
	for _, n := range a.NetAddrNetworks {
		cm.addrParsers[n] = a.ParseNetAddr
	}

	cm.maddrParsers[a.ProtocolName] = a.ConvertMultiaddr
}

var tcpAddrSpec = &NetCodec{
	ProtocolName:     "tcp",
	NetAddrNetworks:  []string{"tcp", "tcp4", "tcp6"},
	ParseNetAddr:     parseTcpNetAddr,
	ConvertMultiaddr: parseBasicNetMaddr,
}

var udpAddrSpec = &NetCodec{
	ProtocolName:     "udp",
	NetAddrNetworks:  []string{"udp", "udp4", "udp6"},
	ParseNetAddr:     parseUdpNetAddr,
	ConvertMultiaddr: parseBasicNetMaddr,
}

var ip4AddrSpec = &NetCodec{
	ProtocolName:     "ip4",
	NetAddrNetworks:  []string{"ip4"},
	ParseNetAddr:     parseIpNetAddr,
	ConvertMultiaddr: parseBasicNetMaddr,
}

var ip6AddrSpec = &NetCodec{
	ProtocolName:     "ip6",
	NetAddrNetworks:  []string{"ip6"},
	ParseNetAddr:     parseIpNetAddr,
	ConvertMultiaddr: parseBasicNetMaddr,
}

var ipnetAddrSpec = &NetCodec{
	ProtocolName:    "ip+net",
	NetAddrNetworks: []string{"ip+net"},
	ParseNetAddr:    parseIpPlusNetAddr,
	ConvertMultiaddr: func(ma.Multiaddr) (net.Addr, error) {
		return nil, fmt.Errorf("converting ip+net multiaddr not supported")
	},
}

func (cm *CodecMap) getAddrParser(net string) (FromNetAddrFunc, error) {
	cm.lk.Lock()
	defer cm.lk.Unlock()

	parser, ok := cm.addrParsers[net]
	if !ok {
		return nil, fmt.Errorf("unknown network %v", net)
	}
	return parser, nil
}

func (cm *CodecMap) getMaddrParser(name string) (ToNetAddrFunc, error) {
	cm.lk.Lock()
	defer cm.lk.Unlock()
	p, ok := cm.maddrParsers[name]
	if !ok {
		return nil, fmt.Errorf("network not supported: %s", name)
	}

	return p, nil
}
