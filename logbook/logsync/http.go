package logsync

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/qri-io/qri/auth/key"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/logbook"
	"github.com/qri-io/qri/profile"
	"github.com/qri-io/qri/repo"
	reporef "github.com/qri-io/qri/repo/ref"
)

// httpClient is the request side of doing dsync over HTTP
type httpClient struct {
	URL string
}

// compile time assertion that httpClient is a remote
// httpClient exists to satisfy the Remote interface on the client side
var _ remote = (*httpClient)(nil)

func (c *httpClient) addr() string {
	return c.URL
}

func (c *httpClient) put(ctx context.Context, author profile.Author, ref dsref.Ref, r io.Reader) error {
	log.Debugw("httpClient.put", "ref", ref)
	u, err := url.Parse(c.URL)
	if err != nil {
		return fmt.Errorf("invalid logsync client url: %w", err)
	}
	q := u.Query()
	// TODO(B5): we need the old serialization format here b/c logsync checks the
	// profileID matches the author. Migrate to a new system for validating who-can-push-what
	// using keystore & UCANs, then switch this to standard reference string serialization
	q.Set("ref", ref.LegacyProfileIDString())
	u.RawQuery = q.Encode()

	req, err := http.NewRequest("PUT", u.String(), r)
	if err != nil {
		return err
	}
	req = req.WithContext(ctx)

	if err := addAuthorHTTPHeaders(req.Header, author); err != nil {
		return err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK {
		if errmsg, err := ioutil.ReadAll(res.Body); err == nil {
			return fmt.Errorf(string(errmsg))
		}
		return err
	}

	return nil
}

func (c *httpClient) get(ctx context.Context, author profile.Author, ref dsref.Ref) (profile.Author, io.Reader, error) {
	log.Debugw("httpClient.get", "ref", ref)
	u, err := url.Parse(c.URL)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid logsync client url: %w", err)
	}
	q := u.Query()
	// TODO(b5): remove initID for backwards compatiblity. get doesn't rely on the
	// ProfileID field, but the other end of the wire may error if we send an InitID
	// field
	ref.InitID = ""
	q.Set("ref", ref.String())
	u.RawQuery = q.Encode()

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, nil, err
	}
	req = req.WithContext(ctx)

	if err := addAuthorHTTPHeaders(req.Header, author); err != nil {
		log.Debugf("addAuthorHTTPHeaders error=%q", err)
		return nil, nil, err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Debugf("http.DefaultClient.Do error=%q", err)
		return nil, nil, err
	}

	if res.StatusCode != http.StatusOK {
		log.Debugf("httpClient.get statusCode=%d", res.StatusCode)
		if errmsg, err := ioutil.ReadAll(res.Body); err == nil {
			return nil, nil, fmt.Errorf(string(errmsg))
		}
		return nil, nil, err
	}

	sender, err := senderFromHTTPHeaders(res.Header)
	if err != nil {
		return nil, nil, err
	}

	return sender, res.Body, nil
}

func (c *httpClient) del(ctx context.Context, author profile.Author, ref dsref.Ref) error {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s?ref=%s", c.URL, ref), nil)
	if err != nil {
		return err
	}
	req = req.WithContext(ctx)

	if err := addAuthorHTTPHeaders(req.Header, author); err != nil {
		return err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK {
		if errmsg, err := ioutil.ReadAll(res.Body); err == nil {
			return fmt.Errorf(string(errmsg))
		}
	}
	return err
}

func addAuthorHTTPHeaders(h http.Header, author profile.Author) error {
	h.Set("ID", author.AuthorID())
	h.Set("username", author.Username())
	pubKey, err := key.EncodePubKeyB64(author.AuthorPubKey())
	if err != nil {
		return err
	}
	h.Set("PubKey", pubKey)
	return nil
}

func senderFromHTTPHeaders(h http.Header) (profile.Author, error) {
	pub, err := key.DecodeB64PubKey(h.Get("PubKey"))
	if err != nil {
		return nil, fmt.Errorf("decoding public key: %s", err)
	}

	return profile.NewAuthor(h.Get("ID"), pub, h.Get("username")), nil
}

// HTTPHandler exposes a Dsync remote over HTTP by exposing a HTTP handler
// that interlocks with methods exposed by httpClient
func HTTPHandler(lsync *Logsync) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sender, err := senderFromHTTPHeaders(r.Header)
		if err != nil {
			log.Debugf("senderFromHTTPHeaders error=%q", err)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}

		switch r.Method {
		case "PUT":
			ref, err := dsref.Parse(r.FormValue("ref"))
			if err != nil {
				log.Debugf("PUT dsref.Parse error=%q", err)
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(err.Error()))
				return
			}
			if err := lsync.put(r.Context(), sender, ref, r.Body); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(err.Error()))
				return
			}
			r.Body.Close()

			addAuthorHTTPHeaders(w.Header(), lsync.Author())
			return
		case "GET":
			ref, err := dsref.Parse(r.FormValue("ref"))
			if err != nil {
				log.Debugf("GET dsref.Parse error=%q", err)
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(err.Error()))
				return
			}

			receiver, r, err := lsync.get(r.Context(), sender, ref)
			if err != nil {
				log.Debugf("GET error=%q ref=%q", err, ref)
				// TODO (ramfox): implement a robust error response strategy
				if errors.Is(err, logbook.ErrNotFound) {
					w.WriteHeader(http.StatusNotFound)
					w.Write([]byte(err.Error()))
					return
				}
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(err.Error()))
				return
			}
			addAuthorHTTPHeaders(w.Header(), receiver)
			io.Copy(w, r)
			return
		case "DELETE":
			ref, err := repo.ParseDatasetRef(r.FormValue("ref"))
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(err.Error()))
				return
			}

			if err = lsync.del(r.Context(), sender, reporef.ConvertToDsref(ref)); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(err.Error()))
				return
			}

			addAuthorHTTPHeaders(w.Header(), lsync.Author())
			return
		default:
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`not found`))
			return
		}
	}
}
