package api

import (
	"fmt"
	"net"
	"net/http"
	"time"

	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	"github.com/qri-io/qri/config"
)

// StartServer interprets info from config to start an API server
func StartServer(c *config.API, s *http.Server) error {
	if !c.Enabled {
		return nil
	}

	addr, err := ma.NewMultiaddr(c.Address)
	if err != nil {
		return err
	}

	var listener net.Listener

	if !c.ServeRemoteTraffic {
		// if we're not serving remote traffic, strip off any address details other
		// than a raw TCP address
		portAddr := config.DefaultAPIPort
		ma.ForEach(addr, func(comp ma.Component) bool {
			if comp.Protocol().Code == ma.P_TCP {
				portAddr = comp.Value()
				return false
			}
			return true
		})

		// use a raw listener on a local TCP socket
		listener, err = net.Listen("tcp", fmt.Sprintf(":%s", portAddr))
		if err != nil {
			return err
		}
	} else {
		mal, err := manet.Listen(addr)
		if err != nil {
			return err
		}
		listener = manet.NetListener(mal)
	}

	return s.Serve(listener)
}

// HTTPSRedirect listens over TCP on addr, redirecting HTTP requests to https
func HTTPSRedirect(addr string) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return
	}

	log.Infof("TCP Port %s is available, redirecting traffic to https", addr)

	srv := &http.Server{
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("Connection", "close")
			url := "https://" + req.Host + req.URL.String()
			http.Redirect(w, req, url, http.StatusMovedPermanently)
		}),
	}
	srv.Serve(ln)
}
