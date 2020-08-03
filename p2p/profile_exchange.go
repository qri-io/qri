package p2p

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
)

// ProfileExchangeProtocolID is the protocol id for the profile exchange service
const ProfileExchangeProtocolID = "/qri/profile/0.1.0"

// ProfileTimeout is the length of time we will wait for a response in a profile
// exchange
const ProfileTimeout = time.Minute * 2

// ProfileExchangeService manages the profile exchange. This exchange should happen
// whenever a node connects to a new peer
type ProfileExchangeService struct {
	Host               host.Host
	Repo               repo.Repo
	pub                event.Publisher
	mutex              *sync.Mutex
	inProcessExchanges map[peer.ID]bool
}

// NewQriProfileExchangeService creates an profile exchange service and adds
// the `ProfileExchangeHandler` to the host
func NewQriProfileExchangeService(h host.Host, r repo.Repo, p event.Publisher) *ProfileExchangeService {
	ps := &ProfileExchangeService{h, r, p, &sync.Mutex{}, map[peer.ID]bool{}}
	h.SetStreamHandler(ProfileExchangeProtocolID, ps.ProfileExchangeHandler)
	return ps
}

// ProfileExchangeHandler listens for profile exchange requests, only waiting
// as long as the ProfileTimeout. It takes any profile that comes off the stream,
// saving it to the profile store & sends back its own profile
func (ps *ProfileExchangeService) ProfileExchangeHandler(s network.Stream) {
	id := s.Conn().RemotePeer()
	log.Debug("exchanging profiles with %q", id)
	exchangeAlreadyInProcess := ps.IsExchangeInProcess(id)
	if exchangeAlreadyInProcess {
		log.Debug("exchange is already in process")
		return
	}
	ps.RecordInProcessExchange(id)
	defer ps.MarkExchangeAsDone(id)
	ps.pub.Publish(context.Background(), event.ETP2PProfileExchangeRequestRecieved, id)
	pubFailure := func() {
		ps.pub.Publish(context.Background(), event.ETP2PProfileExchangeFailed, id)
	}
	defer pubFailure()

	ws := WrapStream(s)
	errCh := make(chan error, 1)
	defer close(errCh)
	timer := time.NewTimer(ProfileTimeout)
	defer timer.Stop()

	go func() {
		select {
		case <-timer.C:
			log.Debug("profile exchange timeout")
		case err, ok := <-errCh:
			if ok {
				log.Debug(err)
			} else {
				log.Error("profile exchange failed without error")
			}
		}
		s.Reset()
	}()

	pro, err := receiveProfile(ws)
	if err != nil {
		errCh <- err
		return
	}

	myPro, err := ps.Repo.Profile()
	if err != nil {
		errCh <- err
		return
	}

	if err := sendProfile(ws, myPro); err != nil {
		errCh <- err
		return
	}
	if err := ps.Repo.Profiles().PutProfile(pro); err != nil {
		errCh <- err
		return
	}
	pubFailure = func() {}
	ps.pub.Publish(context.Background(), event.ETP2PProfileExchangeSuccess, pro.PeerIDs[0])
}

// ProfileExchangeResult is a result of a ping attempt, either an RTT or an error.
type ProfileExchangeResult struct {
	Profile *profile.Profile
	Error   error
}

// ProfileExchange kicks off an exhange of profiles. The node opens a stream to
// the given peer, wraps the stream for easier encoding/decoding, sends its
// profile over the stream, and recieves the peer's profile. Finally, it adds
// that peer's profile to the profile store.
func (ps *ProfileExchangeService) ProfileExchange(ctx context.Context, p peer.ID) error {
	log.Debug("exchanging profiles with %q", p)
	exchangeAlreadyInProcess := ps.IsExchangeInProcess(p)
	if exchangeAlreadyInProcess {
		log.Debug("exchange is already in process")
		return nil
	}
	ps.pub.Publish(context.Background(), event.ETP2PProfileExchangeRequestSent, p)

	ps.RecordInProcessExchange(p)
	defer ps.MarkExchangeAsDone(p)

	pubFailure := func() {
		ps.pub.Publish(context.Background(), event.ETP2PProfileExchangeFailed, p)
	}
	defer pubFailure()

	resCh := ps.profileExchange(ctx, p)
	res := <-resCh
	if res.Error != nil {
		log.Errorf("error exchanging profile with peer %q: %s", p, res.Error)
		return fmt.Errorf("error exchanging profile with peer %q: %s", p, res.Error)
	}
	if err := ps.Repo.Profiles().PutProfile(res.Profile); err != nil {
		log.Errorf("error putting peer %q profile in store: %s", p, err)
		return fmt.Errorf("error putting peer %q profile in store: %s", p, err)
	}
	log.Debug("exchanged profile information with %q", p)

	pubFailure = func() {}
	ps.pub.Publish(context.Background(), event.ETP2PProfileExchangeSuccess, p)
	return nil
}

// profileExchanges talks over the qri protocol to exchange profile information from a peer
func (ps *ProfileExchangeService) profileExchange(ctx context.Context, p peer.ID) <-chan ProfileExchangeResult {
	s, err := ps.Host.NewStream(ctx, p, ProfileExchangeProtocolID)
	if err != nil {
		ch := make(chan ProfileExchangeResult, 1)
		ch <- ProfileExchangeResult{Error: err}
		close(ch)
		return ch
	}

	ws := WrapStream(s)

	ctx, cancel := context.WithCancel(ctx)

	out := make(chan ProfileExchangeResult)
	go func() {
		defer close(out)
		defer cancel()

		for ctx.Err() == nil {
			var res ProfileExchangeResult
			before := time.Now()
			myPro, err := ps.Repo.Profile()
			if err != nil {
				res.Error = err
				out <- res
				return
			}

			res.Error = sendProfile(ws, myPro)
			if err != nil {
				out <- res
				return
			}

			res.Profile, res.Error = receiveProfile(ws)
			RTT := time.Since(before)

			// canceled, ignore everything.
			if ctx.Err() != nil {
				return
			}

			// No error, record the RTT.
			if res.Error == nil {
				ps.Host.Peerstore().RecordLatency(p, RTT)
			}

			select {
			case out <- res:
			case <-ctx.Done():
				return
			}
		}
	}()
	go func() {
		// forces the profile exchange to abort.
		<-ctx.Done()
		ws.stream.Reset()
	}()

	return out
}

func sendProfile(ws *WrappedStream, pro *profile.Profile) error {
	pod, err := pro.Encode()
	if err != nil {
		log.Errorf("error encoding profile.Profile to config.ProfilePod: %s", err)
		return fmt.Errorf("error encoding profile.Profile to config.ProfilePod: %s", err)
	}
	if err := ws.enc.Encode(&pod); err != nil {
		log.Errorf("error encoding profile to wrapped stream: %s", err)
		return fmt.Errorf("error encoding profile to wrapped stream: %s", err)
	}
	if err := ws.w.Flush(); err != nil {
		log.Errorf("error flushing stream: %s", err)
		return fmt.Errorf("error flushing stream: %s", err)
	}
	return nil
}

func receiveProfile(ws *WrappedStream) (*profile.Profile, error) {
	pod := &config.ProfilePod{}
	if err := ws.dec.Decode(&pod); err != nil {
		log.Errorf("error decoding config.ProfilePod from wrapped stream: %s", err)
		return nil, fmt.Errorf("error decoding config.ProfilePod from wrapped stream: %s", err)
	}
	pro := &profile.Profile{}
	if err := pro.Decode(pod); err != nil {
		log.Errorf("error decoding Profile from config.ProfilePod: %s", err)
		return nil, fmt.Errorf("error decoding Profile from config.ProfilePod: %s", err)
	}
	return pro, nil
}

// IsExchangeInProcess checks to see if a change is in process
func (ps *ProfileExchangeService) IsExchangeInProcess(id peer.ID) bool {
	ps.mutex.Lock()
	_, ok := ps.inProcessExchanges[id]
	ps.mutex.Unlock()
	return ok
}

// RecordInProcessExchange adds the id to the in process exchanges
func (ps *ProfileExchangeService) RecordInProcessExchange(id peer.ID) {
	ps.mutex.Lock()
	ps.inProcessExchanges[id] = true
	ps.mutex.Unlock()
}

// MarkExchangeAsDone removes the id from the in process exchanges
func (ps *ProfileExchangeService) MarkExchangeAsDone(id peer.ID) {
	ps.mutex.Lock()
	delete(ps.inProcessExchanges, id)
	ps.mutex.Unlock()
}
