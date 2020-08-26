package logsync

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"

	crypto "github.com/libp2p/go-libp2p-core/crypto"
	host "github.com/libp2p/go-libp2p-core/host"
	net "github.com/libp2p/go-libp2p-core/network"
	peer "github.com/libp2p/go-libp2p-core/peer"
	protocol "github.com/libp2p/go-libp2p-core/protocol"
	"github.com/qri-io/dag/dsync/p2putil"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/identity"
	"github.com/qri-io/qri/repo"
	reporef "github.com/qri-io/qri/repo/ref"
)

const (
	// LogsyncProtocolID is the dsyc p2p Protocol Identifier
	LogsyncProtocolID = protocol.ID("/qri/logsync")
	// LogsyncServiceTag tags the type & version of the dsync service
	LogsyncServiceTag = "qri/logsync/0.1.1-dev"
	// default value to give logsync peer connections in connmanager, one hunnit
	logsyncSupportValue = 100
)

var (
	// mtGet identifies the "put" message type, a client pushing a log to a remote
	mtPut = p2putil.MsgType("put")
	// mtGet identifies the "get" message type, a request for a log
	mtGet = p2putil.MsgType("get")
	// mtDel identifies the "del" message type, a request to remove a log
	mtDel = p2putil.MsgType("del")
)

type p2pClient struct {
	remotePeerID peer.ID
	*p2pHandler
}

// assert at compile time that p2pClient implements DagSyncable
var _ remote = (*p2pClient)(nil)

func (c *p2pClient) addr() string {
	return c.remotePeerID.Pretty()
}

func (c *p2pClient) put(ctx context.Context, author identity.Author, ref dsref.Ref, r io.Reader) (err error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}

	headers := []string{
		"phase", "request",
		"ref", ref.String(),
	}
	headers, err = addAuthorP2PHeaders(headers, author)
	if err != nil {
		return err
	}
	msg := p2putil.NewMessage(c.host.ID(), mtPut, data).WithHeaders(headers...)
	_, err = c.sendMessage(ctx, msg, c.remotePeerID)
	return err
}

func (c *p2pClient) get(ctx context.Context, author identity.Author, ref dsref.Ref) (sender identity.Author, data io.Reader, err error) {
	headers := []string{
		"phase", "request",
		"ref", ref.String(),
	}
	headers, err = addAuthorP2PHeaders(headers, author)
	if err != nil {
		return nil, nil, err
	}

	msg := p2putil.NewMessage(c.host.ID(), mtGet, nil).WithHeaders(headers...)

	res, err := c.sendMessage(ctx, msg, c.remotePeerID)
	if err != nil {
		return nil, nil, err
	}

	sender, err = authorFromP2PHeaders(res)
	return sender, bytes.NewReader(res.Body), err
}

func (c *p2pClient) del(ctx context.Context, author identity.Author, ref dsref.Ref) error {
	headers := []string{
		"phase", "request",
		"ref", ref.String(),
	}
	headers, err := addAuthorP2PHeaders(headers, author)
	if err != nil {
		return err
	}

	msg := p2putil.NewMessage(c.host.ID(), mtDel, nil).WithHeaders(headers...)
	_, err = c.sendMessage(ctx, msg, c.remotePeerID)
	return err
}

func addAuthorP2PHeaders(h []string, author identity.Author) ([]string, error) {
	pkb, err := author.AuthorPubKey().Bytes()
	if err != nil {
		return nil, err
	}
	pubKey := base64.StdEncoding.EncodeToString(pkb)

	return append(h, "author_id", author.AuthorID(), "pub_key", pubKey, "author_name", author.AuthorName()), nil
}

func authorFromP2PHeaders(msg p2putil.Message) (identity.Author, error) {
	data, err := base64.StdEncoding.DecodeString(msg.Header("pub_key"))
	if err != nil {
		return nil, err
	}

	pub, err := crypto.UnmarshalPublicKey(data)
	if err != nil {
		return nil, fmt.Errorf("decoding public key: %s", err)
	}

	return identity.NewAuthor(msg.Header("author_id"), pub, msg.Header("author_name")), nil
}

// p2pHandler implements logsync as a libp2p protocol handler
type p2pHandler struct {
	logsync  *Logsync
	host     host.Host
	handlers map[p2putil.MsgType]p2putil.HandlerFunc
}

// newp2pHandler creates a p2p remote stream handler from a dsync.Remote
func newp2pHandler(logsync *Logsync, host host.Host) *p2pHandler {
	c := &p2pHandler{logsync: logsync, host: host}
	c.handlers = map[p2putil.MsgType]p2putil.HandlerFunc{
		mtPut: c.HandlePut,
		mtGet: c.HandleGet,
		mtDel: c.HandleDel,
	}

	go host.SetStreamHandler(LogsyncProtocolID, c.LibP2PStreamHandler)
	return c
}

// LibP2PStreamHandler provides remote access over p2p
func (c *p2pHandler) LibP2PStreamHandler(s net.Stream) {
	c.handleStream(p2putil.WrapStream(s), nil)
}

// HandlePut requests a new send session from the remote, which will return
// a delta manifest of blocks the remote needs and a session id that must
// be sent with each block
func (c *p2pHandler) HandlePut(ws *p2putil.WrappedStream, msg p2putil.Message) (hangup bool) {
	if msg.Header("phase") == "request" {
		ctx := context.Background()
		author, err := authorFromP2PHeaders(msg)
		if err != nil {
			return true
		}

		ref, err := dsref.Parse(msg.Header("ref"))
		if err != nil {
			return true
		}

		if err = c.logsync.put(ctx, author, ref, bytes.NewReader(msg.Body)); err != nil {
			return true
		}

		headers := []string{
			"phase", "response",
		}
		headers, err = addAuthorP2PHeaders(headers, c.logsync.Author())
		if err != nil {
			return true
		}

		res := msg.WithHeaders(headers...)
		if err := ws.SendMessage(res); err != nil {
			return true
		}
	}
	return true
}

// HandleGet places a block on the remote
func (c *p2pHandler) HandleGet(ws *p2putil.WrappedStream, msg p2putil.Message) (hangup bool) {
	if msg.Header("phase") == "request" {
		ctx := context.Background()
		author, err := authorFromP2PHeaders(msg)
		if err != nil {
			return true
		}

		ref, err := repo.ParseDatasetRef(msg.Header("ref"))
		if err != nil {
			return true
		}

		sender, r, err := c.logsync.get(ctx, author, reporef.ConvertToDsref(ref))
		if err != nil {
			return true
		}

		data, err := ioutil.ReadAll(r)
		if err != nil {
			return true
		}

		headers := []string{
			"phase", "response",
		}
		headers, err = addAuthorP2PHeaders(headers, sender)
		if err != nil {
			return true
		}

		res := msg.WithHeaders(headers...).Update(data)
		if err := ws.SendMessage(res); err != nil {
			return true
		}
	}

	return true
}

// HandleDel asks the remote for a manifest specified by the root ID of a DAG
func (c *p2pHandler) HandleDel(ws *p2putil.WrappedStream, msg p2putil.Message) (hangup bool) {
	if msg.Header("phase") == "request" {
		ctx := context.Background()
		author, err := authorFromP2PHeaders(msg)
		if err != nil {
			return true
		}

		ref, err := repo.ParseDatasetRef(msg.Header("ref"))
		if err != nil {
			return true
		}

		if err = c.logsync.del(ctx, author, reporef.ConvertToDsref(ref)); err != nil {
			return true
		}

		res := msg.WithHeaders("phase", "response")
		if err := ws.SendMessage(res); err != nil {
			return true
		}
	}
	return true
}

// sendMessage opens a stream & sends a message to a peer id
func (c *p2pHandler) sendMessage(ctx context.Context, msg p2putil.Message, pid peer.ID) (p2putil.Message, error) {
	s, err := c.host.NewStream(ctx, pid, LogsyncProtocolID)
	if err != nil {
		return p2putil.Message{}, fmt.Errorf("error opening stream: %s", err.Error())
	}
	defer s.Close()

	// now that we have a confirmed working connection
	// tag this peer as supporting the qri protocol in the connection manager
	// rem.host.ConnManager().TagPeer(pid, logsyncSupportKey, logsyncSupportValue)

	ws := p2putil.WrapStream(s)
	replies := make(chan p2putil.Message)
	go c.handleStream(ws, replies)
	if err := ws.SendMessage(msg); err != nil {
		return p2putil.Message{}, err
	}

	reply := <-replies
	return reply, nil
}

// handleStream is a loop which receives and handles messages
// When Message.HangUp is true, it exits. This will close the stream
// on one of the sides. The other side's receiveMessage() will error
// with EOF, thus also breaking out from the loop.
func (c *p2pHandler) handleStream(ws *p2putil.WrappedStream, replies chan p2putil.Message) {
	for {
		// Loop forever, receiving messages until the other end hangs up
		// or something goes wrong
		msg, err := ws.ReceiveMessage()

		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			// oplog.Debugf("error receiving message: %s", err.Error())
			break
		}

		if replies != nil {
			go func() { replies <- msg }()
		}

		handler, ok := c.handlers[msg.Type]
		if !ok {
			break
		}

		// call the handler, if it returns true, we hangup
		if handler(ws, msg) {
			break
		}
	}

	ws.Close()
}
