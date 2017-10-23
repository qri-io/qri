package discovery

import (
	"context"
	"errors"
	"io"
	"io/ioutil"
	golog "log"
	"net"
	"sync"
	"time"

	pstore "gx/ipfs/QmQMQ2RUjnaEEX8ybmrhuFFGhAwPjyL1Eo6ZoJGD7aAccM/go-libp2p-peerstore"
	ma "gx/ipfs/QmSWLfmj5frN9xVLMMN846dMDriy5wN5jeghUm7aTW3DAG/go-multiaddr"
	logging "gx/ipfs/QmSpJByNKFX1sCsHBEp3R73FL4NF6FnQTEGyNAXHm2GS52/go-log"
	manet "gx/ipfs/QmVCNGTyD4EkvNYaAp253uMQ9Rjsjy2oGMvcdJJUoVRfja/go-multiaddr-net"
	"gx/ipfs/QmWSvDKkcno2UyDg13rUBwWfhRsdj7uR3daAq57VoG5QeN/mdns"
	"gx/ipfs/QmZcUPvPhD1Xvk6mwijYF8AfR3mG31S1YsEfHG4khrFPRr/go-libp2p-peer"
	"gx/ipfs/QmbzbRyd22gcW92U1rA2yKagB3myMYhk45XBknJ49F9XWJ/go-libp2p-host"
)

var log = logging.Logger("mdns")

const ServiceTag = "_ipfs-discovery._udp"

type Service interface {
	io.Closer
	RegisterNotifee(Notifee)
	UnregisterNotifee(Notifee)
}

type Notifee interface {
	HandlePeerFound(pstore.PeerInfo)
}

type mdnsService struct {
	server  *mdns.Server
	service *mdns.MDNSService
	host    host.Host

	lk       sync.Mutex
	notifees []Notifee
	interval time.Duration
}

func getDialableListenAddrs(ph host.Host) ([]*net.TCPAddr, error) {
	var out []*net.TCPAddr
	for _, addr := range ph.Addrs() {
		na, err := manet.ToNetAddr(addr)
		if err != nil {
			continue
		}
		tcp, ok := na.(*net.TCPAddr)
		if ok {
			out = append(out, tcp)
		}
	}
	if len(out) == 0 {
		return nil, errors.New("failed to find good external addr from peerhost")
	}
	return out, nil
}

func NewMdnsService(ctx context.Context, peerhost host.Host, interval time.Duration) (Service, error) {

	// TODO: dont let mdns use logging...
	golog.SetOutput(ioutil.Discard)

	var ipaddrs []net.IP
	port := 4001

	addrs, err := getDialableListenAddrs(peerhost)
	if err != nil {
		log.Warning(err)
	} else {
		port = addrs[0].Port
		for _, a := range addrs {
			ipaddrs = append(ipaddrs, a.IP)
		}
	}

	myid := peerhost.ID().Pretty()

	info := []string{myid}
	service, err := mdns.NewMDNSService(myid, ServiceTag, "", "", port, ipaddrs, info)
	if err != nil {
		return nil, err
	}

	// Create the mDNS server, defer shutdown
	server, err := mdns.NewServer(&mdns.Config{Zone: service})
	if err != nil {
		return nil, err
	}

	s := &mdnsService{
		server:   server,
		service:  service,
		host:     peerhost,
		interval: interval,
	}

	go s.pollForEntries(ctx)

	return s, nil
}

func (m *mdnsService) Close() error {
	return m.server.Shutdown()
}

func (m *mdnsService) pollForEntries(ctx context.Context) {

	ticker := time.NewTicker(m.interval)
	for {
		select {
		case <-ticker.C:
			entriesCh := make(chan *mdns.ServiceEntry, 16)
			go func() {
				for entry := range entriesCh {
					m.handleEntry(entry)
				}
			}()

			log.Debug("starting mdns query")
			qp := &mdns.QueryParam{
				Domain:  "local",
				Entries: entriesCh,
				Service: ServiceTag,
				Timeout: time.Second * 5,
			}

			err := mdns.Query(qp)
			if err != nil {
				log.Error("mdns lookup error: ", err)
			}
			close(entriesCh)
			log.Debug("mdns query complete")
		case <-ctx.Done():
			log.Debug("mdns service halting")
			return
		}
	}
}

func (m *mdnsService) handleEntry(e *mdns.ServiceEntry) {
	log.Debugf("Handling MDNS entry: %s:%d %s", e.AddrV4, e.Port, e.Info)
	mpeer, err := peer.IDB58Decode(e.Info)
	if err != nil {
		log.Warning("Error parsing peer ID from mdns entry: ", err)
		return
	}

	if mpeer == m.host.ID() {
		log.Debug("got our own mdns entry, skipping")
		return
	}

	maddr, err := manet.FromNetAddr(&net.TCPAddr{
		IP:   e.AddrV4,
		Port: e.Port,
	})
	if err != nil {
		log.Warning("Error parsing multiaddr from mdns entry: ", err)
		return
	}

	pi := pstore.PeerInfo{
		ID:    mpeer,
		Addrs: []ma.Multiaddr{maddr},
	}

	m.lk.Lock()
	for _, n := range m.notifees {
		go n.HandlePeerFound(pi)
	}
	m.lk.Unlock()
}

func (m *mdnsService) RegisterNotifee(n Notifee) {
	m.lk.Lock()
	m.notifees = append(m.notifees, n)
	m.lk.Unlock()
}

func (m *mdnsService) UnregisterNotifee(n Notifee) {
	m.lk.Lock()
	found := -1
	for i, notif := range m.notifees {
		if notif == n {
			found = i
			break
		}
	}
	if found != -1 {
		m.notifees = append(m.notifees[:found], m.notifees[found+1:]...)
	}
	m.lk.Unlock()
}
