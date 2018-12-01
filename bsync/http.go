package bsync

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/qri-io/qri/manifest"
)

// HTTPRemote implents the Remote interface via HTTP POST requests
type HTTPRemote struct {
	URL string
}

// ReqSession initiates a send session
func (rem *HTTPRemote) ReqSession(mfst *manifest.Manifest) (sid string, diff *manifest.Manifest, err error) {
	buf := &bytes.Buffer{}
	if err = json.NewEncoder(buf).Encode(mfst); err != nil {
		return
	}

	req, err := http.NewRequest("POST", rem.URL, buf)
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}

	if res.StatusCode != http.StatusOK {
		err = fmt.Errorf("remote repsonse: %d", res.StatusCode)
		return
	}

	sid = res.Header.Get("sid")
	diff = &manifest.Manifest{}
	err = json.NewDecoder(res.Body).Decode(diff)

	return
}

// PutBlock sends a block over HTTP to a remote source
func (rem *HTTPRemote) PutBlock(sid, hash string, data []byte) Response {
	url := fmt.Sprintf("%s?sid=%s&hash=%s", rem.URL, sid, hash)
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(data))
	if err != nil {
		return Response{
			Hash:   hash,
			Status: StatusErrored,
			Err:    err,
		}
	}
	req.Header.Set("Content-Type", "application/octet-stream")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return Response{
			Hash:   hash,
			Status: StatusErrored,
			Err:    err,
		}
	}

	if res.StatusCode != http.StatusOK {
		return Response{
			Hash:   hash,
			Status: StatusRetry,
		}
	}

	return Response{
		Hash:   hash,
		Status: StatusOk,
	}
}
