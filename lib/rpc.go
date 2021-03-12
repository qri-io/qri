package lib

import (
	"context"
	"encoding/gob"
	"encoding/json"
	"io/ioutil"
	stdlog "log"
	"net/rpc"

	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr-net"
)

func init() {
	// We don't use the log package, and the net/rpc package spits out some complaints b/c
	// a few methods don't conform to the proper signature (comment this out & run 'qri connect' to see errors)
	// so we're disabling the log package for now. This is potentially very stupid.
	// TODO (b5): remove dep on net/rpc package entirely
	stdlog.SetOutput(ioutil.Discard)

	// Fields like dataset.Structure.Schema contain data of arbitrary types,
	// registering with the gob package prevents errors when sending them
	// over net/rpc calls.
	gob.Register(json.RawMessage{})
	gob.Register([]interface{}{})
	gob.Register(map[string]interface{}{})
}

// Receivers returns a slice of CoreRequests that defines the full local
// API of lib methods
func Receivers(inst *Instance) []Methods {
	return []Methods{
		NewRemoteMethods(inst),
		NewPeerMethods(inst),
	}
}

// ServeRPC checks for a configured RPC port, and registers a listener if so
func (inst *Instance) ServeRPC(ctx context.Context) {
	cfg := inst.cfg
	if !cfg.RPC.Enabled || cfg.RPC.Address == "" {
		return
	}
	maAddress := cfg.RPC.Address
	addr, err := ma.NewMultiaddr(maAddress)
	if err != nil {
		log.Errorf("cannot start RPC: error parsing RPC address %s: %w", maAddress, err.Error())
	}

	mal, err := manet.Listen(addr)
	if err != nil {
		log.Infof("RPC listen on address %d error: %w", cfg.RPC.Address, err)
		return
	}
	listener := manet.NetListener(mal)

	for _, rcvr := range Receivers(inst) {
		if err := rpc.Register(rcvr); err != nil {
			log.Errorf("cannot start RPC: error registering RPC receiver %s: %w", rcvr.CoreRequestsName(), err.Error())
			return
		}
	}

	go func() {
		<-ctx.Done()
		log.Info("closing RPC")
		listener.Close()
	}()

	rpc.Accept(listener)
	return
}
