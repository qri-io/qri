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
	"github.com/qri-io/qri/profile"
)

var (
	// ErrNoLogsync indicates no logsync pointer has been allocated where one is expected
	ErrNoLogsync = fmt.Errorf("logsync: does not exist")

	log = golog.Logger("logsync")
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

// Hook is a function called at specified points in the sync lifecycle
type Hook func(ctx context.Context, author profile.Author, ref dsref.Ref, l *oplog.Log) error

// Author is the local author of lsync's logbook
func (lsync *Logsync) Author() profile.Author {
	if lsync == nil {
		return nil
	}
	return profile.NewAuthorFromProfile(lsync.book.Owner())
}

// NewPush prepares a Push from the local logsync to a remote destination
// doing a push places a local log on the remote
func (lsync *Logsync) NewPush(ref dsref.Ref, remoteAddr string) (*Push, error) {
	if lsync == nil {
		return nil, ErrNoLogsync
	}

	rem, err := lsync.remoteClient(context.TODO(), remoteAddr)
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
	log.Debugw("NewPull", "ref", ref, "remoteAddr", remoteAddr)

	rem, err := lsync.remoteClient(context.TODO(), remoteAddr)
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

	rem, err := lsync.remoteClient(ctx, remoteAddr)
	if err != nil {
		return err
	}

	if err = rem.del(ctx, lsync.Author(), ref); err != nil {
		return err
	}

	versions, err := lsync.book.Items(ctx, ref, 0, -1, "")
	if err != nil {
		return err
	}

	// record remove as delete of all versions on the remote
	_, _, err = lsync.book.WriteRemoteDelete(ctx, lsync.book.Owner(), ref.InitID, len(versions), remoteAddr)
	return err
}

func (lsync *Logsync) remoteClient(ctx context.Context, remoteAddr string) (rem remote, err error) {
	if strings.HasPrefix(remoteAddr, "http") {
		return &httpClient{URL: remoteAddr}, nil
	}

	// if we're given a logbook authorId, convert it to the active public key ID
	if l, err := lsync.book.Log(ctx, remoteAddr); err == nil {
		remoteAddr = l.Author()
	}

	// if a valid base58 peerID is passed, we're doing a p2p dsync
	if id, err := peer.IDB58Decode(remoteAddr); err == nil {
		if lsync.p2pHandler == nil {
			return nil, fmt.Errorf("no p2p host provided to perform p2p logsync")
		}
		return &p2pClient{remotePeerID: id, p2pHandler: lsync.p2pHandler}, nil
	}
	return nil, fmt.Errorf("unrecognized remote address string: %q", remoteAddr)
}

// remote is an internal interface for methods available on foreign logbooks
// the logsync struct contains the canonical implementation of a remote
// interface. network clients wrap the remote interface with network behaviours,
// using Logsync methods to do the "real work" and echoing that back across the
// client protocol
type remote interface {
	addr() string
	put(ctx context.Context, author profile.Author, ref dsref.Ref, r io.Reader) error
	get(ctx context.Context, author profile.Author, ref dsref.Ref) (sender profile.Author, data io.Reader, err error)
	del(ctx context.Context, author profile.Author, ref dsref.Ref) error
}

// assert at compile-time that Logsync is a remote
var _ remote = (*Logsync)(nil)

// addr is only used by clients. this should never be called
func (lsync *Logsync) addr() string {
	panic("cannot get the address of logsync itself")
}

func (lsync *Logsync) put(ctx context.Context, author profile.Author, ref dsref.Ref, r io.Reader) error {
	if lsync == nil {
		return ErrNoLogsync
	}

	if lsync.pushPreCheck != nil {
		if err := lsync.pushPreCheck(ctx, author, ref, nil); err != nil {
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

	// Get the ref that is in use within the logbook data
	logRef, err := logbook.DsrefAliasForLog(lg)
	if err != nil {
		return err
	}

	// Validate that data in the logbook matches the ref being synced
	if logRef.Username != ref.Username || logRef.Name != ref.Name || logRef.ProfileID != ref.ProfileID {
		return fmt.Errorf("ref contained in log data does not match")
	}

	if lsync.pushFinalCheck != nil {
		if err := lsync.pushFinalCheck(ctx, author, logRef, lg); err != nil {
			return err
		}
	}

	if err := lsync.book.MergeLog(ctx, author.AuthorPubKey(), lg); err != nil {
		return err
	}

	if lsync.pushed != nil {
		if err := lsync.pushed(ctx, author, ref, lg); err != nil {
			log.Debugf("pushed hook error=%q", err)
		}
	}
	return nil
}

func (lsync *Logsync) get(ctx context.Context, author profile.Author, ref dsref.Ref) (profile.Author, io.Reader, error) {
	log.Debugf("logsync.get author.AuthorID=%q ref=%q", author.AuthorID, ref)
	if lsync == nil {
		return nil, nil, ErrNoLogsync
	}

	if lsync.pullPreCheck != nil {
		if err := lsync.pullPreCheck(ctx, author, ref, nil); err != nil {
			log.Debugf("pullPreCheck error=%q author=%q ref=%q", err, author, ref)
			return nil, nil, err
		}
	}

	if _, err := lsync.book.ResolveRef(ctx, &ref); err != nil {
		log.Debugf("book.ResolveRef error=%q ref=%q ", err, ref)
		return nil, nil, err
	}

	l, err := lsync.book.UserDatasetBranchesLog(ctx, ref.InitID)
	if err != nil {
		log.Debugf("book.UserDatasetBranchesLog error=%q initID=%q", err, ref.InitID)
		return lsync.Author(), nil, err
	}

	data, err := lsync.book.LogBytes(l, lsync.book.Owner().PrivKey)
	if err != nil {
		log.Debugf("LogBytes error=%q initID=%q", err, ref.InitID)
		return nil, nil, err
	}

	if lsync.pulled != nil {
		if err := lsync.pulled(ctx, author, ref, l); err != nil {
			log.Debugf("pulled hook error=%q", err)
		}
	}

	return lsync.Author(), bytes.NewReader(data), nil
}

func (lsync *Logsync) del(ctx context.Context, sender profile.Author, ref dsref.Ref) error {
	if lsync == nil {
		return ErrNoLogsync
	}

	if lsync.removePreCheck != nil {
		if err := lsync.removePreCheck(ctx, sender, ref, nil); err != nil {
			return err
		}
	}

	l, err := lsync.book.BranchRef(ctx, ref)
	if err != nil {
		return err
	}

	// eventually access control will dictate which logs can be written by whom.
	// For now we only allow users to delete logs they've written
	// book will need access to a store of public keys before we can verify
	// signatures non-same-senders
	// if err := l.Verify(sender.AuthorPubKey()); err != nil {
	// 	return err
	// }

	root := l
	for {
		p := root.Parent()
		if p == nil {
			break
		}
		root = p
	}

	if err := lsync.book.RemoveLog(ctx, ref); err != nil {
		return err
	}

	if lsync.removed != nil {
		if err := lsync.removed(ctx, sender, ref, nil); err != nil {
			log.Debugf("removed hook error=%q", err)
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
	// eagerly write a push to the logbook. The log the remote receives will include
	// the push operation. If anything goes wrong, rollback the write
	l, rollback, err := p.book.WriteRemotePush(ctx, p.book.Owner(), p.ref.InitID, 1, p.remote.addr())
	if err != nil {
		return err
	}

	data, err := p.book.LogBytes(l, p.book.Owner().PrivKey)
	if err != nil {
		if rollbackErr := rollback(ctx); rollbackErr != nil {
			log.Errorf("rolling back dataset log: %q", rollbackErr)
		}
		return err
	}

	buf := bytes.NewBuffer(data)
	author := profile.NewAuthorFromProfile(p.book.Owner())
	err = p.remote.put(ctx, author, p.ref, buf)
	if err != nil {
		if rollbackErr := rollback(ctx); rollbackErr != nil {
			log.Errorf("rolling back dataset log: %q", rollbackErr)
		}
		return err
	}
	return nil
}

// Pull is a request to fetch a log
type Pull struct {
	book   *logbook.Book
	ref    dsref.Ref
	remote remote

	// set to true to merge these logs into the local store on successful pull
	Merge bool
}

// Do executes the pull
func (p *Pull) Do(ctx context.Context) (*oplog.Log, error) {
	log.Debugw("pull.Do", "ref", p.ref)
	author := profile.NewAuthorFromProfile(p.book.Owner())
	sender, r, err := p.remote.get(ctx, author, p.ref)
	if err != nil {
		return nil, err
	}
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	l := &oplog.Log{}
	if err := l.UnmarshalFlatbufferBytes(data); err != nil {
		return nil, err
	}

	if p.Merge {
		if err := p.book.MergeLog(ctx, sender.AuthorPubKey(), l); err != nil {
			return nil, err
		}
	}

	return l, nil
}
