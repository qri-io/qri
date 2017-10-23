package websocket

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"

	ma "gx/ipfs/QmSWLfmj5frN9xVLMMN846dMDriy5wN5jeghUm7aTW3DAG/go-multiaddr"
	ws "gx/ipfs/QmTDQRD9abCLfdZCjVxuujSVq1X7pkW8eXhaJtgsjH2fnT/websocket"
	manet "gx/ipfs/QmVCNGTyD4EkvNYaAp253uMQ9Rjsjy2oGMvcdJJUoVRfja/go-multiaddr-net"
	tpt "gx/ipfs/QmVpYwkpCJLSLpEY9tUbDQjCVdEVusgibpE9TopF5MPoSS/go-libp2p-transport"
	mafmt "gx/ipfs/QmYjJnSTfXWhYL2cV1xFphPqjqowJqH7ZKLA1As8QrPHbn/mafmt"
)

var WsProtocol = ma.Protocol{
	Code:  477,
	Name:  "ws",
	VCode: ma.CodeToVarint(477),
}

var WsFmt = mafmt.And(mafmt.TCP, mafmt.Base(WsProtocol.Code))

var WsCodec = &manet.NetCodec{
	NetAddrNetworks:  []string{"websocket"},
	ProtocolName:     "ws",
	ConvertMultiaddr: ConvertWebsocketMultiaddrToNetAddr,
	ParseNetAddr:     ParseWebsocketNetAddr,
}

func init() {
	err := ma.AddProtocol(WsProtocol)
	if err != nil {
		panic(fmt.Errorf("error registering websocket protocol: %s", err))
	}

	manet.RegisterNetCodec(WsCodec)
}

func ConvertWebsocketMultiaddrToNetAddr(maddr ma.Multiaddr) (net.Addr, error) {
	_, host, err := manet.DialArgs(maddr)
	if err != nil {
		return nil, err
	}

	a := &ws.Addr{
		URL: &url.URL{
			Host: host,
		},
	}
	return a, nil
}

func ParseWebsocketNetAddr(a net.Addr) (ma.Multiaddr, error) {
	wsa, ok := a.(*ws.Addr)
	if !ok {
		return nil, fmt.Errorf("not a websocket address")
	}

	tcpaddr, err := net.ResolveTCPAddr("tcp", wsa.Host)
	if err != nil {
		return nil, err
	}

	tcpma, err := manet.FromNetAddr(tcpaddr)
	if err != nil {
		return nil, err
	}

	wsma, err := ma.NewMultiaddr("/ws")
	if err != nil {
		return nil, err
	}

	return tcpma.Encapsulate(wsma), nil
}

type WebsocketTransport struct{}

func (t *WebsocketTransport) Matches(a ma.Multiaddr) bool {
	return WsFmt.Matches(a)
}

func (t *WebsocketTransport) Dialer(_ ma.Multiaddr, opts ...tpt.DialOpt) (tpt.Dialer, error) {
	return &dialer{}, nil
}

type dialer struct{}

func parseMultiaddr(a ma.Multiaddr) (string, error) {
	_, host, err := manet.DialArgs(a)
	if err != nil {
		return "", err
	}

	return "ws://" + host, nil
}

func (d *dialer) Dial(raddr ma.Multiaddr) (tpt.Conn, error) {
	return d.DialContext(context.Background(), raddr)
}

func (d *dialer) DialContext(ctx context.Context, raddr ma.Multiaddr) (tpt.Conn, error) {
	wsurl, err := parseMultiaddr(raddr)
	if err != nil {
		return nil, err
	}

	wscon, err := ws.Dial(wsurl, "", "http://127.0.0.1:0/")
	if err != nil {
		return nil, err
	}

	mnc, err := manet.WrapNetConn(wscon)
	if err != nil {
		return nil, err
	}

	return &wsConn{
		Conn: mnc,
	}, nil
}

func (d *dialer) Matches(a ma.Multiaddr) bool {
	return WsFmt.Matches(a)
}

type wsConn struct {
	manet.Conn
	t tpt.Transport
}

func (c *wsConn) Transport() tpt.Transport {
	return c.t
}

type listener struct {
	manet.Listener

	incoming chan *conn

	tpt tpt.Transport
}

type conn struct {
	*ws.Conn

	done func()
}

func (c *conn) Close() error {
	c.done()
	return c.Conn.Close()
}

func (t *WebsocketTransport) Listen(a ma.Multiaddr) (tpt.Listener, error) {
	list, err := manet.Listen(a)
	if err != nil {
		return nil, err
	}

	tlist := t.wrapListener(list)

	u, err := url.Parse("ws://" + list.Addr().String())
	if err != nil {
		return nil, err
	}

	s := &ws.Server{
		Handler: tlist.handleWsConn,
		Config:  ws.Config{Origin: u},
	}

	go http.Serve(list.NetListener(), s)

	return tlist, nil
}

func (t *WebsocketTransport) wrapListener(l manet.Listener) *listener {
	return &listener{
		Listener: l,
		incoming: make(chan *conn),
		tpt:      t,
	}
}

func (l *listener) handleWsConn(s *ws.Conn) {
	ctx, cancel := context.WithCancel(context.Background())
	s.PayloadType = ws.BinaryFrame

	l.incoming <- &conn{
		Conn: s,
		done: cancel,
	}

	// wait until conn gets closed, otherwise the handler closes it early
	<-ctx.Done()
}

func (l *listener) Accept() (tpt.Conn, error) {
	c, ok := <-l.incoming
	if !ok {
		return nil, fmt.Errorf("listener is closed")
	}

	mnc, err := manet.WrapNetConn(c)
	if err != nil {
		return nil, err
	}

	return &wsConn{
		Conn: mnc,
		t:    l.tpt,
	}, nil
}

func (l *listener) Multiaddr() ma.Multiaddr {
	wsma, err := ma.NewMultiaddr("/ws")
	if err != nil {
		panic(err)
	}

	return l.Listener.Multiaddr().Encapsulate(wsma)
}

var _ tpt.Transport = (*WebsocketTransport)(nil)
