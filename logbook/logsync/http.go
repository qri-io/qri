package logsync

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	crypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/identity"
	"github.com/qri-io/qri/logbook"
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

func (c *httpClient) put(ctx context.Context, author identity.Author, r io.Reader) error {
	req, err := http.NewRequest("PUT", c.URL, r)
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

func (c *httpClient) get(ctx context.Context, author identity.Author, ref dsref.Ref) (identity.Author, io.Reader, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s?ref=%s", c.URL, ref), nil)
	if err != nil {
		return nil, nil, err
	}
	req = req.WithContext(ctx)

	if err := addAuthorHTTPHeaders(req.Header, author); err != nil {
		return nil, nil, err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, nil, err
	}
	if res.StatusCode != http.StatusOK {
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

func (c *httpClient) del(ctx context.Context, author identity.Author, ref dsref.Ref) error {
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

func addAuthorHTTPHeaders(h http.Header, author identity.Author) error {
	h.Set("ID", author.AuthorID())

	pubByteStr, err := author.AuthorPubKey().Bytes()
	if err != nil {
		return err
	}
	h.Set("PubKey", base64.StdEncoding.EncodeToString(pubByteStr))
	return nil
}

func senderFromHTTPHeaders(h http.Header) (identity.Author, error) {
	data, err := base64.StdEncoding.DecodeString(h.Get("PubKey"))
	if err != nil {
		return nil, err
	}

	pub, err := crypto.UnmarshalPublicKey(data)
	if err != nil {
		return nil, fmt.Errorf("decoding public key: %s", err)
	}

	return identity.NewAuthor(h.Get("ID"), pub), nil
}

// HTTPHandler exposes a Dsync remote over HTTP by exposing a HTTP handler
// that interlocks with methods exposed by httpClient
func HTTPHandler(lsync *Logsync) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sender, err := senderFromHTTPHeaders(r.Header)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}

		switch r.Method {
		case "PUT":
			if err := lsync.put(r.Context(), sender, r.Body); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(err.Error()))
				return
			}
			r.Body.Close()

			addAuthorHTTPHeaders(w.Header(), lsync.Author())
		case "GET":
			ref, err := repo.ParseDatasetRef(r.FormValue("ref"))
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(err.Error()))
				return
			}

			receiver, r, err := lsync.get(r.Context(), sender, reporef.ConvertToDsref(ref))
			if err != nil {
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
		}
	}
}
