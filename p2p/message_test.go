package p2p

import (
	"context"
	"testing"
)

func TestPing(t *testing.T) {
	ntwk, err := NewTestNetwork()
	if err != nil {
		t.Errorf("error creating network: %s", err.Error())
		return
	}

	a, b := ntwk[0], ntwk[1]
	connectNodes(context.Background(), t, ntwk)

	for i := 1; i <= 10; i++ {
		ping := &Message{
			Phase: MpRequest,
			Type:  MtPing,
		}
		pong, err := a.SendMessage(b.Identity, ping)
		if err != nil {
			t.Errorf("ping %d response error: %s", i, err.Error())
			return
		}
		if pong.Phase != MpResponse {
			t.Errorf("ping %d repsonse should have phase type response, got: %s", i, pong.Phase)
		}
		if pong.Type != MtPing {
			t.Errorf("ping %d response should have message type ping. got: %s", i, pong.Type.String())
		}
	}
}
