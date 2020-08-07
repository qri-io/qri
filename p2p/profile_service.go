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
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
)

// ProfileProtocolID is the protocol id for the profile exchange service
const ProfileProtocolID = "/qri/profile/0.1.0"

// ProfileTimeout is the length of time we will wait for a response in a profile
// exchange
const ProfileTimeout = time.Minute * 2

// ProfileService manages the profile exchange. This exchange should happen
// whenever a node connects to a new peer
type ProfileService struct {
	Host    host.Host
	Repo    repo.Repo
	pub     event.Publisher
	peersMu *sync.Mutex
	peers   map[peer.ID]chan struct{}
}

// NewQriProfileService creates an profile exchange service and adds
// the `ProfileHandler` to the host
func NewQriProfileService(r repo.Repo, p event.Publisher) *ProfileService {
	ps := &ProfileService{
		// Host:    h,
		Repo:    r,
		pub:     p,
		peersMu: &sync.Mutex{},
		peers:   map[peer.ID]chan struct{}{},
	}
	// h.SetStreamHandler(ProfileProtocolID, ps.ProfileHandler)
	return ps
}

// StartProfileService adds a host and a profile handler to the host
// to the profile service
func (ps *ProfileService) StartProfileService(h host.Host) {
	ps.Host = h
	h.SetStreamHandler(ProfileProtocolID, ps.ProfileHandler)
}

// ProfileHandler listens for profile requests
// it sends it's node's profile on the given stream
// whenever a request comes in
func (ps *ProfileService) ProfileHandler(s network.Stream) {

	var (
		err error
		pro *profile.Profile
	)

	ctx := context.Background()

	p := s.Conn().RemotePeer()

	defer func() {
		helpers.FullClose(s)
		if err != nil {
			ps.pub.Publish(ctx, event.ETP2PProfileSendFailed, p)
			return
		}
		ps.pub.Publish(ctx, event.ETP2PProfileSendSuccess, p)
	}()

	log.Debug("%s received a profile request from %s %s", ProfileProtocolID, p, s.Conn().RemoteMultiaddr())

	if err := ps.pub.Publish(ctx, event.ETP2PProfileRequestReceived, p); err != nil {
		log.Errorf("%s publishing event %s", ProfileProtocolID, err)
	}

	pro, err = ps.Repo.Profile()
	if err != nil {
		log.Errorf("%s error getting this node's profile: %s", ProfileProtocolID, err)
		return
	}

	err = sendProfile(s, pro)
	if err != nil {
		log.Errorf("%s error sending profile to %s: %s", ProfileProtocolID, p, err)
		return
	}
}

// // ProfileResult is a result of a ping attempt, either an RTT or an error.
// type ProfileResult struct {
// 	Profile *profile.Profile
// 	Error   error
// 	Wait    chan struct{}
// }

// ProfileRequest requests a profile from a specific peer
func (ps *ProfileService) ProfileRequest(ctx context.Context, pid peer.ID) {

	protocols, err := ps.Host.Peerstore().GetProtocols(pid)
	if err == nil {
		// if we successfully get the protocols, see if
		// the peer speaks the qri profile protocol
		speaksProfileService := false
		for _, prot := range protocols {
			if prot == ProfileProtocolID {
				speaksProfileService = true
				break
			}
		}
		if !speaksProfileService {
			log.Debugf("peer %q does not speak qri profile protocol", pid)
			return
		}
	}
	<-ps.ProfileWait(ctx, pid)
	return
}

// ProfileWait checks to see if a request to this peer is already in progress
// if not, it sends a profile request to the peer
// either way it returns a channel to wait on until the profile request has
// been completed
func (ps *ProfileService) ProfileWait(ctx context.Context, p peer.ID) <-chan struct{} {
	log.Debug("%s initiating peer profile request to %s", ProfileProtocolID, p)
	ps.peersMu.Lock()
	wait, found := ps.peers[p]
	ps.peersMu.Unlock()

	if found {
		log.Debug("%s profile request to %s already in progress", ProfileProtocolID, p)
		return wait
	}

	ps.peersMu.Lock()
	defer ps.peersMu.Unlock()

	wait, found = ps.peers[p]

	if !found {
		wait = make(chan struct{})
		ps.peers[p] = wait

		ps.pub.Publish(ctx, event.ETP2PProfileRequestSent, p)

		go ps.profileRequest(ctx, p, wait)
	}

	return wait
}

// profileRequest requests creates a stream, adds the ProfileProtocol
// then it handles the profile response from the peer
func (ps *ProfileService) profileRequest(ctx context.Context, p peer.ID, signal chan struct{}) {
	var err error

	defer func() {
		close(signal)
		if err != nil {
			ps.pub.Publish(ctx, event.ETP2PProfileReceiveFailed, p)
			return
		}
		ps.pub.Publish(ctx, event.ETP2PProfileReceiveSuccess, p)
	}()

	s, err := ps.Host.NewStream(ctx, p, ProfileProtocolID)
	if err != nil {
		log.Errorf("error opening profile stream to %q: %s", p, err)
		return
	}

	ps.receiveAndStoreProfile(ctx, s)

	return
}

// receiveAndStoreProfile takes a stream, receives a profile off the stream,
// and stores it in the Repo's ProfileStore
func (ps *ProfileService) receiveAndStoreProfile(ctx context.Context, s network.Stream) {
	defer func() {
		go helpers.FullClose(s)
	}()

	log.Debug("%s received profile message from %s %s", s.Protocol(), s.Conn().RemotePeer(), s.Conn().RemoteMultiaddr())

	pro, err := receiveProfile(s)
	if err != nil {
		log.Errorf("%s error reading profile message from %s: %s", s.Protocol(), s.Conn().RemotePeer(), err)
		return
	}

	ps.Repo.Profiles().PutProfile(pro)
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
