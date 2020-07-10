package p2ptest

import (
	"context"
	"testing"
)

// Ensure that when we use ConnectNodes, we are creating a basic connection
// between two nodes
// - we have connections to each peer
// - we have the addrs of each peer
// - we have a tag on each peer
// - we have the protocols each peer supports
func TestConnectNodes(t *testing.T) {
	ctx := context.Background()
	f := NewTestNodeFactory(NewTestableNode)
	testNodes, err := NewTestDirNetwork(ctx, f)
	if err != nil {
		t.Error(err)
		return
	}

	if err := ConnectNodes(ctx, testNodes); err != nil {
		t.Error(err)
	}

	for _, node := range testNodes {
		// test that each conn has a connection to at least one peer id
		pid := node.SimpleAddrInfo().ID
		for _, rnode := range testNodes {
			rpid := rnode.SimpleAddrInfo().ID
			// dont need to check for connections to self
			if pid == rpid {
				continue
			}
			protos, err := node.Host().Peerstore().SupportsProtocols(rpid, string(TestQriProtocolID))
			if err != nil {
				t.Errorf("node %s, error getting %s's protocols", pid, rpid)
			}
			if len(protos) == 0 {
				t.Errorf("node %s does not have a record that node %s can communicate over the testQri protocol", pid, rpid)
			}
			conns := node.Host().Network().ConnsToPeer(rpid)
			if len(conns) == 0 {
				t.Errorf("node %s has no connections to node %s", pid, rpid)
			}
			addrs := node.Host().Peerstore().Addrs(rpid)
			if len(addrs) == 0 {
				t.Errorf("node %s has no addrs for node %s", pid, rpid)
			}
			tag := node.Host().ConnManager().GetTagInfo(rpid)
			if tag == nil {
				t.Errorf("node %s has not tag info on node %s", pid, rpid)
			}
		}
	}
}

// Ensure that when we use ConnectQriNodes:
// - we have connections to each peer
// - we have the addrs of each peer
// - we have a tag on each peer
// - we have the protocols each peer supports
// - we have a record of if the peer supports the qri protocol
// - we have tagged the connection in the conn manager
// - we have a profile of that peer
func TestConnectQriNodes(t *testing.T) {
	ctx := context.Background()
	f := NewTestNodeFactory(NewTestableNode)
	testNodes, err := NewTestDirNetwork(ctx, f)
	if err != nil {
		t.Error(err)
		return
	}

	if err := ConnectNodes(ctx, testNodes); err != nil {
		t.Error(err)
	}

	if err := ConnectQriNodes(ctx, testNodes); err != nil {
		t.Error(err)
	}

	for _, node := range testNodes {
		// test that each conn has a connection to at least one peer id
		pid := node.SimpleAddrInfo().ID
		for _, rnode := range testNodes {
			rpid := rnode.SimpleAddrInfo().ID
			// dont need to check for connections to self
			if pid == rpid {
				continue
			}
			protos, err := node.Host().Peerstore().SupportsProtocols(rpid, string(TestQriProtocolID))
			if err != nil {
				t.Errorf("node %s, error getting %s's protocols", pid, rpid)
			}
			if len(protos) == 0 {
				t.Errorf("node %s does not have a record that node %s can communicate over the testQri protocol", pid, rpid)
			}
			conns := node.Host().Network().ConnsToPeer(rpid)
			if len(conns) == 0 {
				t.Errorf("node %s has no connections to node %s", pid, rpid)
			}
			addrs := node.Host().Peerstore().Addrs(rpid)
			if len(addrs) == 0 {
				t.Errorf("node %s has no addrs for node %s", pid, rpid)
			}
			tag := node.Host().ConnManager().GetTagInfo(rpid)
			if tag == nil {
				t.Errorf("node %s does has not tag info on node %s", pid, rpid)
			}
			if tag != nil {
				tagVal := tag.Tags[TestQriConnManagerTag]
				if tagVal != TestQriConnManagerValue {
					t.Errorf("node %s tag value for %s incorrect. expected: %d, got: %d", pid, rpid, TestQriConnManagerValue, tagVal)
				}
			}
			supports, err := node.Host().Peerstore().Get(rpid, string(TestQriSupportKey))
			if err != nil {
				t.Errorf("node %s error getting %s support from peerstore: %s", pid, rpid, err)
			}
			_, ok := supports.(bool)
			if !ok {
				t.Errorf("node %s support flag for %s is not bool", pid, rpid)
			}
		}
	}
}
