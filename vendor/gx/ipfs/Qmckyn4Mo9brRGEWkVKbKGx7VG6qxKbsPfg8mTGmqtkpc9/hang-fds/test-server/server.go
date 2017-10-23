package main

import (
	"errors"
	"fmt"
	"io"
	"os"

	ma "gx/ipfs/QmSWLfmj5frN9xVLMMN846dMDriy5wN5jeghUm7aTW3DAG/go-multiaddr"
	manet "gx/ipfs/QmVCNGTyD4EkvNYaAp253uMQ9Rjsjy2oGMvcdJJUoVRfja/go-multiaddr-net"
)

var usageText = `usage: %s <multiaddr>
Listen for incoming connections on <multiaddr>, and print when
we have received any. Don't write, or close them.
`

func hang(i int, c manet.Conn) {
	buf := make([]byte, 1024)
	for {
		_, err := c.Read(buf)
		if err == io.EOF {
			fmt.Printf("conn %d closed from %s\n", i, c.RemoteMultiaddr())
			return
		}
		if err != nil {
			fmt.Printf("conn %d read failed: %s\n", i, err)
			return
		}
	}
}

func listenAndHang(a ma.Multiaddr) error {
	l, err := manet.Listen(a)
	if err != nil {
		return err
	}

	fmt.Printf("listening at %s\n", a)

	for i := 0; ; i++ {
		i := i

		c, err := l.Accept()
		if err != nil {
			return err
		}

		fmt.Printf("conn %d accepted from %s\n", i, c.RemoteMultiaddr())
		go hang(i, c)
	}
}

func run(args []string) error {
	m, err := ma.NewMultiaddr(args[0])
	if err != nil {
		return errors.New("<multiaddr> argument must be a valid multiaddr")
	}

	return listenAndHang(m)
}

func main() {
	usageAndExit := func(code int) {
		p := os.Args[0]
		fmt.Printf(usageText, p)
		os.Exit(code)
	}

	for _, arg := range os.Args {
		if arg == "-h" || arg == "--help" {
			usageAndExit(0)
			return
		}
	}
	if len(os.Args) != 2 {
		usageAndExit(-1)
		return
	}

	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(-1)
	}
}
