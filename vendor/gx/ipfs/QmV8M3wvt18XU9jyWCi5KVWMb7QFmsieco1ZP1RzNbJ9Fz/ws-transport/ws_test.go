package websocket

import (
	"bytes"
	"testing"

	ma "gx/ipfs/QmSWLfmj5frN9xVLMMN846dMDriy5wN5jeghUm7aTW3DAG/go-multiaddr"
)

func TestMultiaddrParsing(t *testing.T) {
	addr, err := ma.NewMultiaddr("/ip4/127.0.0.1/tcp/5555/ws")
	if err != nil {
		t.Fatal(err)
	}

	_, err = parseMultiaddr(addr)
	if err != nil {
		t.Fatal(err)
	}
}

func TestWebsocketListen(t *testing.T) {
	zero, err := ma.NewMultiaddr("/ip4/127.0.0.1/tcp/0/ws")
	if err != nil {
		t.Fatal(err)
	}

	tpt := &WebsocketTransport{}
	l, err := tpt.Listen(zero)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	msg := []byte("HELLO WORLD")

	go func() {
		d, _ := tpt.Dialer(nil)
		c, err := d.Dial(l.Multiaddr())
		if err != nil {
			t.Error(err)
			return
		}

		c.Write(msg)
		c.Close()
	}()

	c, err := l.Accept()
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	buf := make([]byte, 32)
	n, err := c.Read(buf)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(buf[:n], msg) {
		t.Fatal("got wrong message", buf[:n], msg)
	}
}
