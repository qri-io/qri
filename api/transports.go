package api

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/qri-io/qri/config"
	"golang.org/x/crypto/acme/autocert"
)

// StartServer interprets info from config to start the server
// if config.TLS == true it'll spin up an https server using LetsEncrypt
// that should work just fine on the raw internet (ie not behind a proxy like nginx etc)
// it'll also redirect http traffic to it's https route counterpart if port 80 is open
func StartServer(c *config.API, s *http.Server) error {
	s.Addr = fmt.Sprintf(fmt.Sprintf(":%d", c.Port))
	if !c.Enabled || c.Port == 0 {
		return nil
	}

	if !c.TLS {
		return s.ListenAndServe()
	}

	// log.Infoln("using https server for url root:", c.UrlRoot)
	certCache := "/tmp/certs"
	key, cert := "", ""

	// LetsEncrypt is good. Thanks LetsEncrypt.
	certManager := autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(c.URLRoot),
		Cache:      autocert.DirCache(certCache),
	}

	// Attempt to boot a port 80 https redirect
	go func() { HTTPSRedirect() }()

	s.TLSConfig = &tls.Config{
		GetCertificate: certManager.GetCertificate,
		// Causes servers to use Go's default ciphersuite preferences,
		// which are tuned to avoid attacks. Does nothing on clients.
		PreferServerCipherSuites: true,
		// Only use curves which have assembly implementations
		CurvePreferences: []tls.CurveID{
			tls.CurveP256,
			tls.X25519,
		},
	}

	return s.ListenAndServeTLS(cert, key)
}

// HTTPSRedirect listens on port 80, redirecting HTTP requests to https
func HTTPSRedirect() {
	ln, err := net.Listen("tcp", ":80")
	if err != nil {
		return
	}

	// log.Infoln("TCP Port 80 is available, redirecting traffic to https")

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
