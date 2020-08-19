package p2p

import (
	"context"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p-core/helpers"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	protocol "github.com/libp2p/go-libp2p-core/protocol"
	"github.com/qri-io/qri/dsref"
)

const (
	// p2pRefResolverTimeout is the length of time we will wait for a
	// RefResolverRequest response before cancelling the context
	// this can potentially be a config option in the future
	p2pRefResolverTimeout = time.Second * 60
	// ResolveRefProtocolID is the protocol on which qri nodes communicate to
	// resolve references
	ResolveRefProtocolID = protocol.ID("/qri/ref/0.1.0")
)

type p2pRefResolver struct {
	node *QriNode
}

type resolveRefRes struct {
	ref    *dsref.Ref
	source string
}

func (rr *p2pRefResolver) ResolveRef(ctx context.Context, ref *dsref.Ref) (string, error) {
	log.Debugf("p2p.ResolveRef ref=%q", ref)
	if rr == nil || rr.node == nil {
		return "", dsref.ErrRefNotFound
	}
	streamCtx, cancel := context.WithTimeout(ctx, p2pRefResolverTimeout)
	defer cancel()

	connectedPids := rr.node.ConnectedQriPeerIDs()
	numReqs := len(connectedPids)
	if numReqs == 0 {
		return "", fmt.Errorf("p2p.ResolveRef error: no connected peers")
	}

	resCh := make(chan resolveRefRes, numReqs)
	for _, pid := range connectedPids {
		go func(pid peer.ID) {
			source := rr.resolveRefRequest(streamCtx, pid, ref)
			resCh <- resolveRefRes{
				ref:    ref,
				source: source,
			}
		}(pid)
	}

	for {
		select {
		case res := <-resCh:
			numReqs--
			if res.ref == nil && numReqs == 0 {
				return "", dsref.ErrRefNotFound
			}
			if res.ref != nil {
				ref = res.ref
				return res.source, nil
			}
		case <-streamCtx.Done():
			log.Debug("p2p.ResolveRef request timed out before resolving ref")
			return "", fmt.Errorf("p2p.ResolveRef request timed out before resolving ref")
		}
	}
}

func (rr *p2pRefResolver) resolveRefRequest(ctx context.Context, pid peer.ID, ref *dsref.Ref) string {
	var (
		err error
		s   network.Stream
	)

	defer func() {
		// helpers.FullClose will close the stream from this end and wait until the other
		// end has also closed
		// This closes the stream not the underlying connection
		go helpers.FullClose(s)
	}()

	s, err = rr.node.Host().NewStream(ctx, pid, ResolveRefProtocolID)
	if err != nil {
		log.Debugf("p2p.ResolveRef - error opening resolve ref stream to peer %q: %s", pid, err)
		return ""
	}

	ref, err = receiveRef(s)
	if err != nil {
		log.Errorf("p2p.ResolveRef - error reading ref message from %q: %s", pid, err)
		return ""
	}
	return s.Conn().RemoteMultiaddr().String()
}

func sendRef(s network.Stream, ref *dsref.Ref) error {
	ws := WrapStream(s)

	if err := ws.enc.Encode(&ref); err != nil {
		return fmt.Errorf("error encoding dsref.Ref to wrapped stream: %s", err)
	}

	if err := ws.w.Flush(); err != nil {
		return fmt.Errorf("error flushing stream: %s", err)
	}

	return nil
}

func receiveRef(s network.Stream) (*dsref.Ref, error) {
	ws := WrapStream(s)
	ref := &dsref.Ref{}
	if err := ws.dec.Decode(ref); err != nil {
		return nil, fmt.Errorf("error decoding dsref.Ref from wrapped stream: %s", err)
	}
	return ref, nil
}

// NewP2PRefResolver creates a resolver backed by a qri node
func (q *QriNode) NewP2PRefResolver() dsref.Resolver {
	return &p2pRefResolver{node: q}
}

// ResolveRefHandler is a handler func that belongs on the QriNode
// it handles request made on the `ResolveRefProtocol`
func (q *QriNode) resolveRefHandler(s network.Stream) {
	if q.localResolver == nil {
		log.Errorf("p2p.ResolverRef - qri node has no local resolver, and so cannot handle ref resolution")
		return
	}
	var (
		err error
		ref *dsref.Ref
	)
	ctx, cancel := context.WithTimeout(context.Background(), p2pRefResolverTimeout)
	defer func() {
		// close the stream, and wait for the other end of the stream to close as well
		// this won't close the underlying connection
		helpers.FullClose(s)
		cancel()
	}()

	p := s.Conn().RemotePeer()
	log.Debugf("p2p.ResolveRef received a ref request from %s %s", p, s.Conn().RemoteMultiaddr())

	// try to resolve this ref locally
	_, err = q.localResolver.ResolveRef(ctx, ref)
	if err != nil {
		log.Debugf("p2p.ResolveRef - error resolving ref locally: %s", err)
		// if there was any error, return a nil ref
		ref = nil
	}

	err = sendRef(s, ref)
	if err != nil {
		log.Debugf("p2p.ResolveRef - error sending ref to %q: %s", p, err)
		return
	}
}
