// Package logsync synchronizes logs between logbooks across networks
package logsync

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	golog "github.com/ipfs/go-log"
	host "github.com/libp2p/go-libp2p-core/host"
	peer "github.com/libp2p/go-libp2p-core/peer"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/logbook"
	"github.com/qri-io/qri/logbook/oplog"
)

var (
	// ErrNoLogsync indicates no logsync pointer has been allocated where one is expected
	ErrNoLogsync = fmt.Errorf("logsync: does not exist")

	logger = golog.Logger("logsync")
)

// Logsync fulfills requests from clients, logsync wraps a logbook.Book, pushing
// and pulling logs from remote sources to its logbook
type Logsync struct {
	book       *logbook.Book
	p2pHandler *p2pHandler

	pushPreCheck   Hook
	pushFinalCheck Hook
	pushed         Hook
	pullPreCheck   Hook
	pulled         Hook
	removePreCheck Hook
	removed        Hook
}

// Options encapsulates runtime configuration for a remote
type Options struct {
	// to send & push over libp2p connections, provide a libp2p host
	Libp2pHost host.Host

	// called before accepting a log, returning an error cancel receiving
	PushPreCheck Hook
	// called after log data has been received, before it's stored in the logbook
	PushFinalCheck Hook
	// called after a log has been merged into the logbook
	Pushed Hook
	// called before a pull is accepted
	PullPreCheck Hook
	// called after a log is pulled
	Pulled Hook
	// called before removing
	RemovePreCheck Hook
	// called after removing
	Removed Hook
}

// New creates a remote from a logbook and optional configuration functions
func New(book *logbook.Book, opts ...func(*Options)) *Logsync {
	o := &Options{}
	for _, opt := range opts {
		opt(o)
	}

	logsync := &Logsync{
		book: book,

		pushPreCheck:   o.PushPreCheck,
		pushFinalCheck: o.PushFinalCheck,
		pushed:         o.Pushed,
		pullPreCheck:   o.PullPreCheck,
		pulled:         o.Pulled,
		removePreCheck: o.RemovePreCheck,
		removed:        o.Removed,
	}

	if o.Libp2pHost != nil {
		logsync.p2pHandler = newp2pHandler(logsync, o.Libp2pHost)
	}

	return logsync
}

// Author is an interface for fetching the ID & public key of a log author
// TODO (b5) - this should be moved into it's own package, and probs just work
// with existing libp2p peer structs
type Author = oplog.Author

// Hook is a function called at specified points in the sync lifecycle
type Hook func(ctx context.Context, author Author, ref dsref.Ref, log *oplog.Log) error

// Author is the local author of lsync's logbook
func (lsync *Logsync) Author() Author {
	if lsync == nil {
		return nil
	}
	return lsync.book.Author()
}

// NewPush prepares a Push from the local logsync to a remote destination
// doing a push places a local log on the remote
func (lsync *Logsync) NewPush(ref dsref.Ref, remoteAddr string) (*Push, error) {
	if lsync == nil {
		return nil, ErrNoLogsync
	}

	rem, err := lsync.remoteClient(remoteAddr)
	if err != nil {
		return nil, err
	}

	return &Push{
		book:   lsync.book,
		remote: rem,
		ref:    ref,
	}, nil
}

// NewPull creates a Pull from the local logsync to a remote destination
// doing a pull fetches a log from the remote to the local logbook
func (lsync *Logsync) NewPull(ref dsref.Ref, remoteAddr string) (*Pull, error) {
	if lsync == nil {
		return nil, ErrNoLogsync
	}

	rem, err := lsync.remoteClient(remoteAddr)
	if err != nil {
		return nil, err
	}

	return &Pull{
		book:   lsync.book,
		remote: rem,
		ref:    ref,
	}, nil
}

// DoRemove asks a remote to remove a log
func (lsync *Logsync) DoRemove(ctx context.Context, ref dsref.Ref, remoteAddr string) error {
	if lsync == nil {
		return ErrNoLogsync
	}

	rem, err := lsync.remoteClient(remoteAddr)
	if err != nil {
		return err
	}

	return rem.del(ctx, lsync.Author(), ref)
}

func (lsync *Logsync) remoteClient(remoteAddr string) (rem remote, err error) {
	if strings.HasPrefix(remoteAddr, "http") {
		return &httpClient{URL: remoteAddr}, nil
	}

	// if we're given a logbook authorId, convert it to the active public key ID
	if l, err := lsync.book.Log(remoteAddr); err == nil {
		remoteAddr = l.Author()
	}

	// if a valid base58 peerID is passed, we're doing a p2p dsync
	if id, err := peer.IDB58Decode(remoteAddr); err == nil {
		if lsync.p2pHandler == nil {
			return nil, fmt.Errorf("no p2p host provided to perform p2p logsync")
		}
		return &p2pClient{remotePeerID: id, p2pHandler: lsync.p2pHandler}, nil
	}
	return nil, fmt.Errorf("unrecognized push address string: %s", remoteAddr)
}

// remove is an internal interface for methods available on foreign logbooks
// the logsync struct contains the canonical implementation of a remote
// interface. network clients wrap the remote interface with network behaviours,
// using Logsync methods to do the "real work" and echoing that back across the
// client protocol
type remote interface {
	put(ctx context.Context, author oplog.Author, r io.Reader) error
	get(ctx context.Context, author oplog.Author, ref dsref.Ref) (sender oplog.Author, data io.Reader, err error)
	del(ctx context.Context, author oplog.Author, ref dsref.Ref) error
}

// assert at compile-time that Logsync is a remote
var _ remote = (*Logsync)(nil)

func (lsync *Logsync) put(ctx context.Context, author oplog.Author, r io.Reader) error {
	if lsync == nil {
		return ErrNoLogsync
	}

	if lsync.pushPreCheck != nil {
		if err := lsync.pushPreCheck(ctx, author, dsref.Ref{}, nil); err != nil {
			return err
		}
	}

	data, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return fmt.Errorf("no data provided to merge")
	}

	lg := &oplog.Log{}
	if err := lg.UnmarshalFlatbufferBytes(data); err != nil {
		return err
	}

	ref, err := logbook.DsrefAliasForLog(lg)
	if err != nil {
		return err
	}

	if lsync.pushFinalCheck != nil {
		if err := lsync.pushFinalCheck(ctx, author, ref, lg); err != nil {
			return err
		}
	}

	if err := lsync.book.MergeLog(ctx, author, lg); err != nil {
		return err
	}

	if lsync.pushed != nil {
		if err := lsync.pushed(ctx, author, ref, lg); err != nil {
			logger.Errorf("pushed hook: %s", err)
		}
	}
	return nil
}

func (lsync *Logsync) get(ctx context.Context, author oplog.Author, ref dsref.Ref) (oplog.Author, io.Reader, error) {
	if lsync == nil {
		return nil, nil, ErrNoLogsync
	}

	if lsync.pullPreCheck != nil {
		if err := lsync.pullPreCheck(ctx, author, ref, nil); err != nil {
			return nil, nil, err
		}
	}

	l, err := lsync.book.HeadRef(ref)
	if err != nil {
		return lsync.Author(), nil, err
	}
	data, err := lsync.book.LogBytes(l)
	if err != nil {
		return nil, nil, err
	}

	if lsync.pulled != nil {
		if err := lsync.pulled(ctx, author, ref, l); err != nil {
			logger.Errorf("pulled hook: %s", err)
		}
	}

	return lsync.Author(), bytes.NewReader(data), nil
}

func (lsync *Logsync) del(ctx context.Context, sender oplog.Author, ref dsref.Ref) error {
	if lsync == nil {
		return ErrNoLogsync
	}

	if lsync.removePreCheck != nil {
		if err := lsync.removePreCheck(ctx, sender, ref, nil); err != nil {
			return err
		}
	}

	if err := lsync.book.RemoveLog(ctx, sender, ref); err != nil {
		return err
	}

	if lsync.removed != nil {
		if err := lsync.removed(ctx, sender, ref, nil); err != nil {
			logger.Errorf("removed hook: %s", err)
		}
	}

	return nil
}

// Push is a request to place a log on a remote
type Push struct {
	ref    dsref.Ref
	book   *logbook.Book
	remote remote
}

// Do executes a push
func (p *Push) Do(ctx context.Context) error {
	log, err := p.book.HeadRef(p.ref)
	if err != nil {
		return err
	}
	data, err := p.book.LogBytes(log)
	if err != nil {
		return err
	}

	buf := bytes.NewBuffer(data)
	return p.remote.put(ctx, p.book.Author(), buf)
}

// Pull is a request to move a log from a remote to the local logsync
type Pull struct {
	book   *logbook.Book
	ref    dsref.Ref
	remote remote
}

// Do executes the pull
func (p *Pull) Do(ctx context.Context) error {
	sender, r, err := p.remote.get(ctx, p.book.Author(), p.ref)
	if err != nil {
		return err
	}
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}

	l := &oplog.Log{}
	if err := l.UnmarshalFlatbufferBytes(data); err != nil {
		return err
	}

	return p.book.MergeLog(ctx, sender, l)
}
