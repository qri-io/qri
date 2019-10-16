// Package logsync synchronizes logs between logbooks across networks
package logsync

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	host "github.com/libp2p/go-libp2p-host"
	peer "github.com/libp2p/go-libp2p-peer"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/logbook"
	"github.com/qri-io/qri/logbook/log"
)

// ErrNoLogsync indicates no logsync pointer has been allocated where one is expected
var ErrNoLogsync = fmt.Errorf("logsync: does not exist")

// Logsync fulfills requests from clients, logsync wraps a logbook.Book, pushing
// and pulling logs from remote sources to its logbook
type Logsync struct {
	book       *logbook.Book
	p2pHandler *p2pHandler

	receiveCheck Hook
	didReceive   Hook
}

// remove is an internal interface for methods available on foreign logbooks
// the logsync struct contains the canonical implementation of a remote
// interface. network clients wrap the remote interface with network behaviours,
// using Logsync methods to do the "real work" and echoing that back across the
// client protocol
type remote interface {
	put(ctx context.Context, author log.Author, r io.Reader) error
	get(ctx context.Context, author log.Author, ref dsref.Ref) (sender log.Author, data io.Reader, err error)
	del(ctx context.Context, author log.Author, ref dsref.Ref) error
}

// assert at compile-time that Logsync is a remote
var _ remote = (*Logsync)(nil)

// Options encapsulates runtime configuration for a remote
type Options struct {
	// ReceiveCheck is called before accepting a log, returning an error from this
	// check will cancel receiving
	ReceiveCheck Hook
	// DidReceive is called after a log has been merged into the logbook
	DidReceive Hook

	// to send & push over libp2p connections, provide a libp2p host
	Libp2pHost host.Host
}

// New creates a remote from a logbook and optional configuration functions
func New(book *logbook.Book, opts ...func(*Options)) *Logsync {
	o := &Options{}
	for _, opt := range opts {
		opt(o)
	}

	logsync := &Logsync{
		book: book,

		receiveCheck: o.ReceiveCheck,
		didReceive:   o.DidReceive,
	}

	if o.Libp2pHost != nil {
		logsync.p2pHandler = newp2pHandler(logsync, o.Libp2pHost)
	}

	return logsync
}

// Hook is a function called at specified points in the sync lifecycle
type Hook func(ctx context.Context, author log.Author, path string) error

// Author is the local author of lsync's logbook
func (lsync *Logsync) Author() log.Author {
	return lsync.book.Author()
}

func (lsync *Logsync) put(ctx context.Context, author log.Author, r io.Reader) error {
	if lsync == nil {
		return ErrNoLogsync
	}

	if lsync.receiveCheck != nil {
		// TODO (b5) - need to populate path
		if err := lsync.receiveCheck(ctx, author, ""); err != nil {
			return err
		}
	}

	data, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}

	if err := lsync.book.MergeLogBytes(ctx, author, data); err != nil {
		return err
	}

	if lsync.didReceive != nil {
		// TODO (b5) - need to populate path
		if err := lsync.didReceive(ctx, author, ""); err != nil {
			return err
		}
	}
	return nil
}

func (lsync *Logsync) get(ctx context.Context, author log.Author, ref dsref.Ref) (log.Author, io.Reader, error) {
	if lsync == nil {
		return nil, nil, ErrNoLogsync
	}

	data, err := lsync.book.LogBytes(ref)
	if err != nil {
		return lsync.Author(), nil, err
	}
	return lsync.Author(), bytes.NewReader(data), nil
}

func (lsync *Logsync) del(ctx context.Context, sender log.Author, ref dsref.Ref) error {
	if lsync == nil {
		return ErrNoLogsync
	}

	return lsync.book.RemoveLog(ctx, sender, ref)
}

// NewPush prepares a Push from the local logsync to a remote destination
// doing a push places a local log on the remote
func (lsync *Logsync) NewPush(ref dsref.Ref, remote string) (*Push, error) {
	rem, err := lsync.getRemote(remote)
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
func (lsync *Logsync) NewPull(ref dsref.Ref, remote string) (*Pull, error) {
	rem, err := lsync.getRemote(remote)
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
func (lsync *Logsync) DoRemove(ctx context.Context, ref dsref.Ref, remote string) error {
	rem, err := lsync.getRemote(remote)
	if err != nil {
		return err
	}

	return rem.del(ctx, lsync.Author(), ref)
}

func (lsync *Logsync) getRemote(remoteAddr string) (rem remote, err error) {
	// if a valid base58 peerID is passed, we're doing a p2p dsync
	if id, err := peer.IDB58Decode(remoteAddr); err == nil {
		if lsync.p2pHandler == nil {
			return nil, fmt.Errorf("no p2p host provided to perform p2p logsync")
		}
		return &p2pClient{remotePeerID: id, p2pHandler: lsync.p2pHandler}, nil
	} else if strings.HasPrefix(remoteAddr, "http") {
		rem = &httpClient{URL: remoteAddr}
	} else {
		return nil, fmt.Errorf("unrecognized push address string: %s", remoteAddr)
	}

	return rem, nil
}

// Push is a request to place a log on a remote
type Push struct {
	ref    dsref.Ref
	book   *logbook.Book
	remote remote
}

// Do executes a push
func (p *Push) Do(ctx context.Context) error {
	data, err := p.book.LogBytes(p.ref)
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

	return p.book.MergeLogBytes(ctx, sender, data)
}
