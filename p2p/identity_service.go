package p2p

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p-core/helpers"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	protocol "github.com/libp2p/go-libp2p-core/protocol"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
)

const (
	// ProfileProtocolID is the protocol id for the profile exchange service
	ProfileProtocolID = protocol.ID("/qri/profile/0.1.0")
	// ProfileTimeout is the length of time we will wait for a response in a
	// profile exchange
	ProfileTimeout = time.Minute * 2
)

var (
	// expectedProtocols is the list of protocols we expect a node that has a
	// profile service to speak
	expectedProtocols = []string{
		string(QriProtocolID),
		string(ProfileProtocolID),
	}
	// ErrPeerNotFound is returned when the profile service cannot find the
	// peer in question
	ErrPeerNotFound = fmt.Errorf("peer not found")
)

// QriIdentityService manages the profile exchange. This exchange should happen
// whenever a node connects to a new peer
type QriIdentityService struct {
	host host.Host
	repo repo.Repo
	pub  event.Publisher
	// profiles is a store that has access to all the profiles we have
	// ever seen on this node
	profiles profile.Store
	// peersMu is the mutext lock for the peers map
	peersMu *sync.Mutex
	// peers is a map of peers to a channel
	// that channel is closed once the profile has been received
	// it only tracks peers we are currently connected to
	peers map[peer.ID]chan struct{}
}

// NewQriIdentityService creates an profile exchange service
func NewQriIdentityService(r repo.Repo, p event.Publisher) *QriIdentityService {
	q := &QriIdentityService{
		repo:     r,
		pub:      p,
		profiles: r.Profiles(),
		peersMu:  &sync.Mutex{},
		peers:    map[peer.ID]chan struct{}{},
	}

	return q
}

// ConnectedQriPeers returns a list of currently connected peers
func (q *QriIdentityService) ConnectedQriPeers() []peer.ID {
	q.peersMu.Lock()
	defer q.peersMu.Unlock()
	peers := []peer.ID{}
	for pid := range q.peers {
		peers = append(peers, pid)
	}
	return peers
}

// HandleQriPeerDisconnect checks if a given peer is a qri peer.
// If so, it will wait until all profile exchanging has finished
// Then remove the peer from the peers map, as well as publish
// that the peer has disconnected.
func (q *QriIdentityService) HandleQriPeerDisconnect(pid peer.ID) {
	q.peersMu.Lock()
	wait, ok := q.peers[pid]
	q.peersMu.Unlock()
	if !ok {
		return
	}
	<-wait

	go func() {
		pro, err := q.profiles.PeerProfile(pid)
		if err != nil {
			log.Debugf("error getting peer's profile. pid=%q err=%q", pid, err)
		}
		if err := q.pub.Publish(context.Background(), event.ETP2PQriPeerDisconnected, pro); err != nil {
			log.Debugf("error publishing ETP2PQriPeerDisconnected event. pid=%q err=%q", pid, err)
		}
	}()

	q.peersMu.Lock()
	delete(q.peers, pid)
	q.peersMu.Unlock()
}

// ConnectedPeerProfile returns a profile if that peer id refers to a peer that
// we have connected to this session.
func (q *QriIdentityService) ConnectedPeerProfile(pid peer.ID) *profile.Profile {
	q.peersMu.Lock()
	_, ok := q.peers[pid]
	q.peersMu.Unlock()
	if !ok {
		return nil
	}
	pro, err := q.profiles.PeerProfile(pid)
	if err != nil {
		log.Debugf("error getting peer profile: pid=%q err=%q", pid, err)
		return nil
	}
	return pro
}

// Start adds a profile handler to the host, retains a local reference to the host
func (q *QriIdentityService) Start(h host.Host) {
	q.host = h
	h.SetStreamHandler(ProfileProtocolID, q.ProfileHandler)
}

// ProfileHandler listens for profile requests
// it sends it's node's profile on the given stream
// whenever a request comes in
func (q *QriIdentityService) ProfileHandler(s network.Stream) {
	var (
		err error
		pro *profile.Profile
	)

	p := s.Conn().RemotePeer()

	defer func() {
		// close the stream, and wait for the other end of the stream to close as well
		// this won't close the underlying connection
		helpers.FullClose(s)
	}()

	log.Debugf("%s received a profile request from %s %s", ProfileProtocolID, p, s.Conn().RemoteMultiaddr())

	pro, err = q.repo.Profile()
	if err != nil {
		log.Debugf("%s error getting this node's profile: %s", ProfileProtocolID, err)
		return
	}

	err = sendProfile(s, pro)
	if err != nil {
		log.Debugf("%s error sending profile to %s: %s", ProfileProtocolID, p, err)
		return
	}
}

// QriIdentityRequest determine if the remote peer speaks the qri protocol
// if it does, it protects the connection and sends a request for the
// QriIdentifyService to get the peer's qri profile information
func (q *QriIdentityService) QriIdentityRequest(ctx context.Context, pid peer.ID) error {
	protocols, err := q.host.Peerstore().SupportsProtocols(pid, expectedProtocols...)
	if err != nil {
		log.Debugf("error examining the protocols for peer %s: %w", pid, err)
		return fmt.Errorf("error examining the protocols for peer %s: %w", pid, err)
	}

	if len(protocols) == len(expectedProtocols) {
		log.Debugf("peer %q does not speak the expected qri protocols", pid)
		return fmt.Errorf("peer %q does not speak the expected qri protocols", pid)
	}

	// protect the connection from pruning
	q.host.ConnManager().Protect(pid, qriSupportKey)
	// get the peer's profile information
	<-q.profileWait(ctx, pid)
	return nil
}

// ProfileWait checks to see if a request to this peer is already in progress
// if not, it sends a profile request to the peer
// either way it returns a channel to wait on until the profile request has
// been completed
func (q *QriIdentityService) profileWait(ctx context.Context, p peer.ID) <-chan struct{} {
	log.Debugf("%s initiating peer profile request to %s", ProfileProtocolID, p)
	q.peersMu.Lock()
	wait, found := q.peers[p]
	q.peersMu.Unlock()

	if found {
		log.Debugf("%s profile request to %s has already occured", ProfileProtocolID, p)
		return wait
	}

	q.peersMu.Lock()
	defer q.peersMu.Unlock()

	wait, found = q.peers[p]

	if !found {
		wait = make(chan struct{})
		q.peers[p] = wait

		go q.profileRequest(ctx, p, wait)
	}

	return wait
}

// profileRequest requests creates a stream, adds the ProfileProtocol
// then it handles the profile response from the peer
func (q *QriIdentityService) profileRequest(ctx context.Context, pid peer.ID, signal chan struct{}) {
	var err error

	defer func() {
		close(signal)
		if err == nil {
			pro, err := q.repo.Profiles().PeerProfile(pid)
			if err != nil {
				log.Debugf("error getting profile from profile store: %s", err)
			}
			go func() {
				if err := q.pub.Publish(ctx, event.ETP2PQriPeerConnected, pro); err != nil {
					log.Debugf("error publishing ETP2PQriPeerConnected event. pid=%q err=%q", pid, err)
				}
			}()
		}
	}()

	s, err := q.host.NewStream(ctx, pid, ProfileProtocolID)
	if err != nil {
		log.Debugf("error opening profile stream to %q: %s", pid, err)
		return
	}

	q.receiveAndStoreProfile(ctx, s)

	return
}

// receiveAndStoreProfile takes a stream, receives a profile off the stream,
// and stores it in the Repo's ProfileStore
func (q *QriIdentityService) receiveAndStoreProfile(ctx context.Context, s network.Stream) {
	defer func() {
		go helpers.FullClose(s)
	}()

	log.Debugf("%s received profile message from %s %s", s.Protocol(), s.Conn().RemotePeer(), s.Conn().RemoteMultiaddr())

	pro, err := receiveProfile(s)
	if err != nil {
		log.Errorf("%s error reading profile message from %s: %s", s.Protocol(), s.Conn().RemotePeer(), err)
		return
	}

	q.repo.Profiles().PutProfile(pro)
	return
}

func sendProfile(s network.Stream, pro *profile.Profile) error {
	ws := WrapStream(s)

	pod, err := pro.Encode()
	if err != nil {
		return fmt.Errorf("error encoding profile.Profile to config.ProfilePod: %s", err)
	}

	if err := ws.enc.Encode(&pod); err != nil {
		return fmt.Errorf("error encoding profile to wrapped stream: %s", err)
	}

	if err := ws.w.Flush(); err != nil {
		return fmt.Errorf("error flushing stream: %s", err)
	}

	return nil
}

func receiveProfile(s network.Stream) (*profile.Profile, error) {
	ws := WrapStream(s)
	pod := &config.ProfilePod{}
	if err := ws.dec.Decode(&pod); err != nil {
		return nil, fmt.Errorf("error decoding config.ProfilePod from wrapped stream: %s", err)
	}
	pro := &profile.Profile{}
	if err := pro.Decode(pod); err != nil {
		return nil, fmt.Errorf("error decoding Profile from config.ProfilePod: %s", err)
	}
	return pro, nil
}
