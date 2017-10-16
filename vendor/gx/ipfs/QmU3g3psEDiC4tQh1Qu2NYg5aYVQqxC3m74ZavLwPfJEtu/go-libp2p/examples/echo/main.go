package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"strings"

	swarm "gx/ipfs/QmNT1JbT5S89ew7kozkHoX5KUG1rfPZvTU3oMDRyJua7rU/go-libp2p-swarm"
	peerstore "gx/ipfs/QmQMQ2RUjnaEEX8ybmrhuFFGhAwPjyL1Eo6ZoJGD7aAccM/go-libp2p-peerstore"
	pstore "gx/ipfs/QmQMQ2RUjnaEEX8ybmrhuFFGhAwPjyL1Eo6ZoJGD7aAccM/go-libp2p-peerstore"
	gologging "gx/ipfs/QmQvJiADDe7JR4m968MwXobTCCzUqQkP87aRHe29MEBGHV/go-logging"
	inet "gx/ipfs/QmRuZnMorqodado1yeTQiv1i9rmtKj29CjPSsBKM7DFXV4/go-libp2p-net"
	net "gx/ipfs/QmRuZnMorqodado1yeTQiv1i9rmtKj29CjPSsBKM7DFXV4/go-libp2p-net"
	ma "gx/ipfs/QmSWLfmj5frN9xVLMMN846dMDriy5wN5jeghUm7aTW3DAG/go-multiaddr"
	golog "gx/ipfs/QmSpJByNKFX1sCsHBEp3R73FL4NF6FnQTEGyNAXHm2GS52/go-log"
	bhost "gx/ipfs/QmU3g3psEDiC4tQh1Qu2NYg5aYVQqxC3m74ZavLwPfJEtu/go-libp2p/p2p/host/basic"
	testutil "gx/ipfs/QmYTzt6uVtDmB5U3iYiA165DQ39xaNLjr8uuDhDtDByXYp/go-testutil"
	peer "gx/ipfs/QmZcUPvPhD1Xvk6mwijYF8AfR3mG31S1YsEfHG4khrFPRr/go-libp2p-peer"
	host "gx/ipfs/QmbzbRyd22gcW92U1rA2yKagB3myMYhk45XBknJ49F9XWJ/go-libp2p-host"
)

// create a 'Host' with a random peer to listen on the given address
func makeBasicHost(listen string, secio bool) (host.Host, error) {
	addr, err := ma.NewMultiaddr(listen)
	if err != nil {
		return nil, err
	}

	ps := pstore.NewPeerstore()
	var pid peer.ID

	if secio {
		ident, err := testutil.RandIdentity()
		if err != nil {
			return nil, err
		}

		ident.PrivateKey()
		ps.AddPrivKey(ident.ID(), ident.PrivateKey())
		ps.AddPubKey(ident.ID(), ident.PublicKey())
		pid = ident.ID()
	} else {
		fakepid, err := testutil.RandPeerID()
		if err != nil {
			return nil, err
		}
		pid = fakepid
	}

	ctx := context.Background()

	// create a new swarm to be used by the service host
	netw, err := swarm.NewNetwork(ctx, []ma.Multiaddr{addr}, pid, ps, nil)
	if err != nil {
		return nil, err
	}

	log.Printf("I am %s/ipfs/%s\n", addr, pid.Pretty())
	return bhost.New(netw), nil
}

func main() {
	golog.SetAllLoggers(gologging.INFO) // Change to DEBUG for extra info
	listenF := flag.Int("l", 0, "wait for incoming connections")
	target := flag.String("d", "", "target peer to dial")
	secio := flag.Bool("secio", false, "enable secio")

	flag.Parse()

	if *listenF == 0 {
		log.Fatal("Please provide a port to bind on with -l")
	}

	listenaddr := fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", *listenF)

	ha, err := makeBasicHost(listenaddr, *secio)
	if err != nil {
		log.Fatal(err)
	}

	// Set a stream handler on host A
	ha.SetStreamHandler("/echo/1.0.0", func(s net.Stream) {
		log.Println("Got a new stream!")
		defer s.Close()
		doEcho(s)
	})

	if *target == "" {
		log.Println("listening for connections")
		select {} // hang forever
	}
	// This is where the listener code ends

	ipfsaddr, err := ma.NewMultiaddr(*target)
	if err != nil {
		log.Fatalln(err)
	}

	pid, err := ipfsaddr.ValueForProtocol(ma.P_IPFS)
	if err != nil {
		log.Fatalln(err)
	}

	peerid, err := peer.IDB58Decode(pid)
	if err != nil {
		log.Fatalln(err)
	}

	tptaddr := strings.Split(ipfsaddr.String(), "/ipfs/")[0]
	// This creates a MA with the "/ip4/ipaddr/tcp/port" part of the target
	tptmaddr, err := ma.NewMultiaddr(tptaddr)
	if err != nil {
		log.Fatalln(err)
	}

	// We need to add the target to our peerstore, so we know how we can
	// contact it
	ha.Peerstore().AddAddr(peerid, tptmaddr, peerstore.PermanentAddrTTL)

	log.Println("opening stream")
	// make a new stream from host B to host A
	// it should be handled on host A by the handler we set above
	s, err := ha.NewStream(context.Background(), peerid, "/echo/1.0.0")
	if err != nil {
		log.Fatalln(err)
	}

	_, err = s.Write([]byte("Hello, world!"))
	if err != nil {
		log.Fatalln(err)
	}

	out, err := ioutil.ReadAll(s)
	if err != nil {
		log.Fatalln(err)
	}

	log.Printf("read reply: %q\n", out)
}

// doEcho reads some data from a stream, writes it back and closes the
// stream.
func doEcho(s inet.Stream) {
	buf := make([]byte, 1024)
	n, err := s.Read(buf)
	if err != nil {
		log.Println(err)
		return
	}

	log.Printf("read request: %q\n", buf[:n])
	_, err = s.Write(buf[:n])
	if err != nil {
		log.Println(err)
		return
	}
}
