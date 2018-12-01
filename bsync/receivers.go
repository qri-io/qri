package bsync

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/qri-io/qri/manifest"

	ipld "gx/ipfs/QmR7TcHkR9nxkUorfi8XMTAMLUK7GiP64TWWBzY3aacc1o/go-ipld-format"
	coreiface "gx/ipfs/QmUJYo4etAQqFfSS2rarFAE97eNGB8ej64YkRT2SmsYD4r/go-ipfs/core/coreapi/interface"
)

// Receivers is a pool of active Receive sessions
type Receivers struct {
	ctx  context.Context
	lng  ipld.NodeGetter
	bapi coreiface.BlockAPI

	lock    sync.Mutex
	pool    map[string]*Receive
	cancels map[string]context.CancelFunc

	TTLDur time.Duration
}

// NewReceivers allocates a Receivers pointer
func NewReceivers(ctx context.Context, lng ipld.NodeGetter, bapi coreiface.BlockAPI) *Receivers {
	return &Receivers{
		ctx:     ctx,
		lng:     lng,
		bapi:    bapi,
		pool:    map[string]*Receive{},
		cancels: map[string]context.CancelFunc{},

		TTLDur: time.Hour * 5,
	}
}

// ReqSession initiates a receive session
func (rs *Receivers) ReqSession(mfst *manifest.Manifest) (sid string, diff *manifest.Manifest, err error) {
	ctx, cancel := context.WithDeadline(rs.ctx, time.Now().Add(rs.TTLDur))
	r, err := NewReceive(ctx, rs.lng, rs.bapi, mfst)
	if err != nil {
		cancel()
		return
	}

	rs.lock.Lock()
	defer rs.lock.Unlock()
	rs.pool[r.sid] = r
	rs.cancels[r.sid] = cancel

	return r.sid, r.diff, nil
}

// PutBlock adds one block in a receive session
func (rs *Receivers) PutBlock(sid, hash string, data []byte) Response {
	r, ok := rs.pool[sid]
	if !ok {
		return Response{
			Hash:   hash,
			Status: StatusErrored,
			Err:    fmt.Errorf("sid not found"),
		}
	}

	res := r.ReceiveBlock(hash, bytes.NewReader(data))

	if res.Status == StatusOk && r.Complete() {
		defer func() {
			rs.lock.Lock()
			rs.cancels[sid]()
			delete(rs.pool, sid)
			rs.lock.Unlock()
		}()
	}

	return res
}

// Handler exposes Receivers over HTTP
func (rs *Receivers) Handler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		mfst := &manifest.Manifest{}
		if err := json.NewDecoder(r.Body).Decode(mfst); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}
		r.Body.Close()

		sid, diff, err := rs.ReqSession(mfst)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}

		w.Header().Set("sid", sid)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(diff)

	case "PUT":
		data, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}

		res := rs.PutBlock(r.FormValue("sid"), r.FormValue("hash"), data)

		if res.Status == StatusErrored {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(res.Err.Error()))
		} else if res.Status == StatusRetry {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(res.Err.Error()))
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}
}
