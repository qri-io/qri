package log

import (
	"fmt"
	"testing"
	"context"
	"math/rand"
	"encoding/json"
)

func TestSimulateLogs(t *testing.T) {
	ctx, done := context.WithCancel(context.Background())
	defer done()

	og := &OpGenerator{ctx: ctx, NoopProb: 95}
	sim := NewSimulation(ctx, 5, og)
	sim.Setup()
	sim.Run(100)

	fmt.Println("ops generated", og.opsGenerated)
	for _, p := range sim.Peers {
		fmt.Printf("peer %d. sent: %d \treceived: %d \tclock: %d \tlogSize: %d\n", p.ID, p.msgsSent, p.msgsReceived, p.Log.Clock(), len(p.Log.Ops))
	}

	for _, p := range sim.Peers {
		fmt.Println()
		state := p.Log.State()
		data, _ := json.Marshal(state)
		fmt.Println(string(data))
	}
}

type Simulation struct {
	ctx context.Context
	Peers []*Peer
}

func NewSimulation(ctx context.Context, peerCount int, gen *OpGenerator) *Simulation {
	s := &Simulation{ctx: ctx, Peers: make([]*Peer, peerCount) }
	for i := 0; i < peerCount; i++ {
		p := &Peer{
			ID: i, 
			Log: Log{}, 
			Inbox: make(chan Op, peerCount),

			ops: gen,
			ticks: make(chan struct{}),
		}
		s.Peers[i] = p
	}

	return s
}

func (s Simulation) Setup() {
	// wire up all peers into a star topology
	for i, a := range s.Peers {
		a.Start(s.ctx)
		for _, b := range s.Peers[i+1:] {
			a.Downstreams = append(a.Downstreams, b.Inbox)
			b.Downstreams = append(b.Downstreams, a.Inbox)
		}
	}
}

func (s Simulation) Run(ticks int) {
	for t := 0; t <= ticks; t++ {
		for _, p := range s.Peers {
			p.Tick(t)
		}
	}

	for _, p := range s.Peers {
		p.Finalize()
	}
}

type OpGenerator struct {
	ctx context.Context
	NoopProb int
	opsGenerated int
}

func (og *OpGenerator) Gen(id int) Op {
	op := Op{}
	i := rand.Intn(100)
	if i > og.NoopProb {
		op = Op{Type: OpTypeDatasetCommit, Author: fmt.Sprintf("%d",id), Note: fmt.Sprintf("op number %d", og.opsGenerated)}
		og.opsGenerated++
	}

	return op
}

type Peer struct {
	ID int
	Log Log
	Inbox chan Op
	Downstreams []chan Op
	
	ops *OpGenerator
	message *Op
	msgsSent int
	msgsReceived int
	ticks chan struct{}
}

func (p *Peer) Start(ctx context.Context) {
	go func() {
		for {
			<- p.ticks
			// fmt.Printf("%d ticked\n", p.ID)
			select {
			case msg := <- p.Inbox:
				p.message = &msg
			case <- ctx.Done():
				return
			}
		}
	}()
}

func (p *Peer) Tick(t int) {
	go func() { p.ticks <- struct{}{} }()

	if p.message != nil {
		p.processMessage()
		return
	}

	op := p.ops.Gen(p.ID)
	if op.Type != OpTypeUnknown {
		p.Log = p.Log.Put(op)
		p.msgsSent++
		for _, ds := range p.Downstreams {
			ds <- op
		}
	}
}

func (p *Peer) Finalize() {
	for len(p.Inbox) > 0 {
		fmt.Println("draining message")
		msg := <- p.Inbox
		p.message = &msg
		p.processMessage()
	}
}

func (p *Peer) processMessage() {
	op := *p.message
	if op.Type != OpTypeUnknown {
		p.msgsReceived++
		p.Log = p.Log.Put(op)
	}
	p.message = nil
}
