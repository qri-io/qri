package p2p

import (
	"encoding/json"
	"time"

	"github.com/qri-io/qri/repo"

	peer "gx/ipfs/QmZoWKhxUmZ2seW4BzX6fJkNR8hh9PsGModr7q171yq2SS/go-libp2p-peer"
)

// MtEvents is a message to announce added / removed datasets to the network
const MtEvents = MsgType("list_events")

// EventsParams encapsulates options for requesting Event logs
type EventsParams struct {
	Limit, Offset int
	Since         time.Time
}

// RequestEventsList fetches a log of events from a peer
func (n *QriNode) RequestEventsList(pid peer.ID, p EventsParams) ([]*repo.Event, error) {
	log.Debugf("%s: RequestEventsList", n.ID)

	if pid == n.ID {
		// requesting self isn't a network operation
		return n.Repo.Events(p.Limit, p.Offset)
	}

	req, err := NewJSONBodyMessage(n.ID, MtEvents, p)
	if err != nil {
		return nil, err
	}

	req = req.WithHeaders("phase", "request")

	replies := make(chan Message)
	n.SendMessage(req, replies, pid)

	res := <-replies
	events := []*repo.Event{}
	err = json.Unmarshal(res.Body, &events)

	return events, err
}

func (n *QriNode) handleEvents(ws *WrappedStream, msg Message) (hangup bool) {
	hangup = true

	switch msg.Header("phase") {
	case "request":
		ep := EventsParams{}
		if err := json.Unmarshal(msg.Body, &ep); err != nil {
			log.Debugf("%s %s", n.ID, err.Error())
			return
		}

		if ep.Limit == 0 || ep.Limit > listMax {
			ep.Limit = listMax
		}

		refs, err := n.Repo.Events(ep.Limit, ep.Offset)
		if err != nil {
			log.Debug(err.Error())
			return
		}

		reply, err := msg.UpdateJSON(refs)
		reply = reply.WithHeaders("phase", "response")
		if err := ws.sendMessage(reply); err != nil {
			log.Debug(err.Error())
			return
		}
	}

	return
}
