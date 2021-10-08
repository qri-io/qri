package websocket

import (
	"bufio"
	"bytes"
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/qri-io/qri/auth/key"
	testkeys "github.com/qri-io/qri/auth/key/test"
	"github.com/qri-io/qri/auth/token"
	"github.com/qri-io/qri/event"
)

func TestWebsocket(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// create key store & add test key
	kd := testkeys.GetKeyData(0)
	ks, err := key.NewMemStore()
	if err != nil {
		t.Fatal(err)
	}
	if err := ks.AddPubKey(context.Background(), kd.KeyID, kd.PrivKey.GetPublic()); err != nil {
		t.Fatal(err)
	}

	// create bus
	bus := event.NewBus(ctx)

	subsCount := bus.NumSubscribers()

	// create Handler
	websocketHandler, err := NewHandler(ctx, bus, ks)
	if err != nil {
		t.Fatal(err)
	}
	wsh := websocketHandler.(*connections)

	// websockets should subscribe the message handler
	if bus.NumSubscribers() != subsCount+1 {
		t.Fatalf("failed to subscribe websocket handlers")
	}

	// add connection
	randIDStr := "test_connection_id_str"
	setIDRand(strings.NewReader(randIDStr))
	connID := newID()
	setIDRand(strings.NewReader(randIDStr))

	wsh.ConnectionHandler(mockWriterAndRequest())
	if _, err := wsh.getConn(connID); err != nil {
		t.Fatal("ConnectionHandler did not create a connection")
	}

	// create a token from a private key
	kd = testkeys.GetKeyData(0)
	tokenStr, err := token.NewPrivKeyAuthToken(kd.PrivKey, kd.KeyID.String(), 0)
	if err != nil {
		t.Fatal(err)
	}
	// upgrade connection w/ valid token
	wsh.subscribeConn(connID, tokenStr)
	proID := kd.KeyID.String()
	gotConnIDs, err := wsh.getConnIDs(proID)
	if err != nil {
		t.Fatal("connections.subscribeConn did not add profileID or conn to subscriptions map")
	}
	if _, ok := gotConnIDs[connID]; !ok {
		t.Fatalf("connections.subscribeConn added incorrect connID to subscriptions map, expected %q, got %q", connID, gotConnIDs)
	}

	// unsubscribe connection via profileID
	wsh.unsubscribeConn(proID, "")
	if _, err := wsh.getConnIDs(proID); err == nil {
		t.Fatal("connections.unsubscribeConn did not remove the profileID from the subscription map")
	}
	wsc, err := wsh.getConn(connID)
	if err != nil {
		t.Fatalf("connection %s not found", connID)
	}
	if wsc.profileID != "" {
		t.Error("connections.unsubscribeConn did not remove the profileID from the conn")
	}

	// remove the connection
	wsh.removeConn(connID)
	if _, err := wsh.getConn(connID); err == nil {
		t.Fatal("connections.removeConn did not remove the connection from the map of conns")
	}
}

func mockWriterAndRequest() (http.ResponseWriter, *http.Request) {
	w := mockHijacker{
		ResponseWriter: httptest.NewRecorder(),
	}

	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Connection", "keep-alive, Upgrade")
	r.Header.Set("Upgrade", "websocket")
	r.Header.Set("Sec-WebSocket-Version", "13")
	r.Header.Set("Sec-WebSocket-Key", "test_key")
	return w, r
}

type mockHijacker struct {
	http.ResponseWriter
}

var _ http.Hijacker = mockHijacker{}

func (mj mockHijacker) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	c, _ := net.Pipe()
	r := bufio.NewReader(strings.NewReader("test_reader"))
	w := bufio.NewWriter(&bytes.Buffer{})
	rw := bufio.NewReadWriter(r, w)
	return c, rw, nil
}
