package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	crypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/qri-io/qri/auth/key"
	"github.com/qri-io/qri/remote/registry"
)

// base64-encoded Test Private Key, decoded in init
var (
	// peerId: QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt
	testPk1  = `CAASpgkwggSiAgEAAoIBAQC/7Q7fILQ8hc9g07a4HAiDKE4FahzL2eO8OlB1K99Ad4L1zc2dCg+gDVuGwdbOC29IngMA7O3UXijycckOSChgFyW3PafXoBF8Zg9MRBDIBo0lXRhW4TrVytm4Etzp4pQMyTeRYyWR8e2hGXeHArXM1R/A/SjzZUbjJYHhgvEE4OZy7WpcYcW6K3qqBGOU5GDMPuCcJWac2NgXzw6JeNsZuTimfVCJHupqG/dLPMnBOypR22dO7yJIaQ3d0PFLxiDG84X9YupF914RzJlopfdcuipI+6gFAgBw3vi6gbECEzcohjKf/4nqBOEvCDD6SXfl5F/MxoHurbGBYB2CJp+FAgMBAAECggEAaVOxe6Y5A5XzrxHBDtzjlwcBels3nm/fWScvjH4dMQXlavwcwPgKhy2NczDhr4X69oEw6Msd4hQiqJrlWd8juUg6vIsrl1wS/JAOCS65fuyJfV3Pw64rWbTPMwO3FOvxj+rFghZFQgjg/i45uHA2UUkM+h504M5Nzs6Arr/rgV7uPGR5e5OBw3lfiS9ZaA7QZiOq7sMy1L0qD49YO1ojqWu3b7UaMaBQx1Dty7b5IVOSYG+Y3U/dLjhTj4Hg1VtCHWRm3nMOE9cVpMJRhRzKhkq6gnZmni8obz2BBDF02X34oQLcHC/Wn8F3E8RiBjZDI66g+iZeCCUXvYz0vxWAQQKBgQDEJu6flyHPvyBPAC4EOxZAw0zh6SF/r8VgjbKO3n/8d+kZJeVmYnbsLodIEEyXQnr35o2CLqhCvR2kstsRSfRz79nMIt6aPWuwYkXNHQGE8rnCxxyJmxV4S63GczLk7SIn4KmqPlCI08AU0TXJS3zwh7O6e6kBljjPt1mnMgvr3QKBgQD6fAkdI0FRZSXwzygx4uSg47Co6X6ESZ9FDf6ph63lvSK5/eue/ugX6p/olMYq5CHXbLpgM4EJYdRfrH6pwqtBwUJhlh1xI6C48nonnw+oh8YPlFCDLxNG4tq6JVo071qH6CFXCIank3ThZeW5a3ZSe5pBZ8h4bUZ9H8pJL4C7yQKBgFb8SN/+/qCJSoOeOcnohhLMSSD56MAeK7KIxAF1jF5isr1TP+rqiYBtldKQX9bIRY3/8QslM7r88NNj+aAuIrjzSausXvkZedMrkXbHgS/7EAPflrkzTA8fyH10AsLgoj/68mKr5bz34nuY13hgAJUOKNbvFeC9RI5g6eIqYH0FAoGAVqFTXZp12rrK1nAvDKHWRLa6wJCQyxvTU8S1UNi2EgDJ492oAgNTLgJdb8kUiH0CH0lhZCgr9py5IKW94OSM6l72oF2UrS6PRafHC7D9b2IV5Al9lwFO/3MyBrMocapeeyaTcVBnkclz4Qim3OwHrhtFjF1ifhP9DwVRpuIg+dECgYANwlHxLe//tr6BM31PUUrOxP5Y/cj+ydxqM/z6papZFkK6Mvi/vMQQNQkh95GH9zqyC5Z/yLxur4ry1eNYty/9FnuZRAkEmlUSZ/DobhU0Pmj8Hep6JsTuMutref6vCk2n02jc9qYmJuD7iXkdXDSawbEG6f5C4MUkJ38z1t1OjA==`
	privKey1 crypto.PrivKey

	testPk2  = `CAASqAkwggSkAgEAAoIBAQDdqbl7nT6hQnTDD+nMkrSLzyoqnx2l+kfF2GN7hZDQGMbh5VgvXyEUifnczUbEIGT/llyOdQmDIvsiGBCMU1T+P1MuhzxSKgblrLtp7yAf6jUgQU6GsbJ5r+MvstG6ds7QqPgKidJL302V0+FMJP6nmpupowDxYQe5GqGJuGwNYBqGTrxqM4FsWNquNPmuE0vCDLqYs2vm6ur2k5RIyTXhnFbpHyO31qsgU5d1dR/Wda0KlyQrS0k3Cmj1foRFGuKJKDJVJ1FTryLAWv9VDSCooQKpWUQ3cUuUSuw9OmTnuvC2xx0IaDAjlh8l+4FRbA+nySVsk82B30MlGYc6jSyDAgMBAAECggEAEhvNhWXBOhddxpnENew+R7Wy8ixxlZ+uwWD+L5cnz3hWtxmvbJ9O6oijGwDCKT+kQKUeBp1VG5t9/LkOkQg1x1eRChoOOYApdBX6cZsResn9cRckvShDNmHCI6FuNNeD6dQD/4hm37/sbLMUks3q5/JfiSpB53ZP1TVxwPiKC0WJriS+dHC6kuiilA0uA+lgOD/w2voqeiFQrjcDu71b3DUulamwq3zt4h2I5pnaOKw7N22k2T9rADS7WbBHVIdd7bxgLkc6EEyho7PT4HqOH15QVS9B4Y4xIYVk0Osqq+uDqSTNEn8SBtL9RE3sO7ygQoKgLL3uvRSXGwP4ISNcYQKBgQD5sBrICL+7+JyAxieirR8kHUS2pfn6/rMrx8lu/fHs6yjOpeHojlKdaCcWLkYn5iahCwayNMFiu/0S9mA3FhNI+nANHbGH/I8RET3EJVPPvdfPhP9YpAXiQYLO5OaSdoKrEyqURDUI3ve+xy/2+JSX4R9q8ovH8c2m7L6gjzlMGQKBgQDjRECMkOxoOxC4i/j3tGI30qz+vYFtpDwz34cEKyKo/tEdmG6PWvm1NlY/GJNvi8fVBxPsIoP1m6ys/842ALYygwm4pV3c3xO8XdrJ8k8jfy6UzbsHXXsoTx9ofqz34IYJ71Pw8MGdwgkwr1qTizKB28E4c22CaAg6yudTcU4Q+wKBgQChh/preq1/z8B/1rIBnfpNhNnVR99HL8t+AUwhkAwY97F4rvxNVPXBe4X95YXhfhVzjgyQ8WxCkdeRku5/9LoZNluTQKh/jzaHFh5dbMCh3vFlAWeoUsSzsSoM6yz3h8/VGRssvEuLJ6QjOf2fywVmlG+c4rjna1leKj7Q5Jdu0QKBgF+5W8bZM/ojBsP0kQUkgUop/pu9jkp0JrdiqyfiU1MDIWlpzweqtgrRvDoPS+pr4dukg4uubg6BZ5XmmSC95AAamXmgjYx+mX15urHc0eCNrT0X+nL7uOgdi4kj8g7mDw8YMy8E+UhNdjl/YpNKyhdQTG5OkA2ha/X3iL/otY0JAoGBAM2E6orKtOiJuLHA+kZfp5BdSNpx5QYGtG+hnPuHspxmFHR8Kj5LgQgJwUsZ6aXtcxPpYOEERAjehJ66CIREHfsy2l1BPdtMPlHIPnnYWSrwtQg0T38VSzhIrFBenOcwg27EGiwPZGccw3JrgyWKRJ7zB5DILlC9306Hz4JNwtGH`
	privKey2 crypto.PrivKey
)

func init() {
	var err error
	privKey1, err = key.DecodeB64PrivKey(testPk1)
	if err != nil {
		panic(err)
	}
	privKey2, err = key.DecodeB64PrivKey(testPk2)
	if err != nil {
		panic(err)
	}
}

func TestProfile(t *testing.T) {
	s := httptest.NewServer(NewRoutes(registry.Registry{Profiles: registry.NewMemProfiles()}))

	p1, err := registry.ProfileFromPrivateKey(&registry.Profile{Username: "b5"}, privKey1)
	if err != nil {
		t.Errorf("error generating profile: %s", err.Error())
		return
	}

	p2, err := registry.ProfileFromPrivateKey(&registry.Profile{Username: "b5"}, privKey2)
	if err != nil {
		t.Errorf("error generating profile: %s", err.Error())
		return
	}

	p1Rename, err := registry.ProfileFromPrivateKey(&registry.Profile{Username: "b6"}, privKey1)
	if err != nil {
		t.Errorf("error generating profile: %s", err.Error())
		return
	}

	type env struct {
		Data *registry.Profile
		Meta struct {
			Code int
		}
	}

	b5 := &registry.Profile{
		ProfileID: "QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt",
		Username:  "b5",
		PublicKey: "CAASpgIwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQC/7Q7fILQ8hc9g07a4HAiDKE4FahzL2eO8OlB1K99Ad4L1zc2dCg+gDVuGwdbOC29IngMA7O3UXijycckOSChgFyW3PafXoBF8Zg9MRBDIBo0lXRhW4TrVytm4Etzp4pQMyTeRYyWR8e2hGXeHArXM1R/A/SjzZUbjJYHhgvEE4OZy7WpcYcW6K3qqBGOU5GDMPuCcJWac2NgXzw6JeNsZuTimfVCJHupqG/dLPMnBOypR22dO7yJIaQ3d0PFLxiDG84X9YupF914RzJlopfdcuipI+6gFAgBw3vi6gbECEzcohjKf/4nqBOEvCDD6SXfl5F/MxoHurbGBYB2CJp+FAgMBAAE=",
	}

	b6 := &registry.Profile{
		ProfileID: "QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt",
		Username:  "b6",
		PublicKey: "CAASpgIwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQC/7Q7fILQ8hc9g07a4HAiDKE4FahzL2eO8OlB1K99Ad4L1zc2dCg+gDVuGwdbOC29IngMA7O3UXijycckOSChgFyW3PafXoBF8Zg9MRBDIBo0lXRhW4TrVytm4Etzp4pQMyTeRYyWR8e2hGXeHArXM1R/A/SjzZUbjJYHhgvEE4OZy7WpcYcW6K3qqBGOU5GDMPuCcJWac2NgXzw6JeNsZuTimfVCJHupqG/dLPMnBOypR22dO7yJIaQ3d0PFLxiDG84X9YupF914RzJlopfdcuipI+6gFAgBw3vi6gbECEzcohjKf/4nqBOEvCDD6SXfl5F/MxoHurbGBYB2CJp+FAgMBAAE=",
	}

	cases := []struct {
		method      string
		contentType string
		profile     *registry.Profile
		resStatus   int
		res         *env
	}{
		{"OPTIONS", "", nil, http.StatusBadRequest, nil},
		{"OPTIONS", "application/json", nil, http.StatusBadRequest, nil},
		{"OPTIONS", "application/json", &registry.Profile{Username: "foo"}, http.StatusNotFound, nil},
		{"POST", "", nil, http.StatusBadRequest, nil},
		{"POST", "application/json", nil, http.StatusBadRequest, nil},
		{"POST", "application/json", &registry.Profile{Username: p1.Username}, http.StatusBadRequest, nil},
		{"POST", "application/json", &registry.Profile{Username: p1.Username, ProfileID: p1.ProfileID}, http.StatusBadRequest, nil},
		{"POST", "application/json", &registry.Profile{Username: p1.Username, ProfileID: p1.ProfileID, Signature: p1.Signature}, http.StatusBadRequest, nil},
		{"POST", "application/json", p1, http.StatusOK, nil},
		{"GET", "application/json", &registry.Profile{Username: b5.Username}, http.StatusOK, &env{Data: b5}},
		{"GET", "application/json", &registry.Profile{Username: "b5"}, http.StatusOK, nil},
		{"GET", "application/json", &registry.Profile{Username: "b6"}, http.StatusNotFound, nil},
		{"GET", "application/json", &registry.Profile{ProfileID: b5.ProfileID}, http.StatusOK, nil},
		{"GET", "application/json", &registry.Profile{ProfileID: "fooooo"}, http.StatusNotFound, nil},
		{"POST", "application/json", p1, http.StatusOK, nil},
		{"POST", "application/json", p2, http.StatusBadRequest, nil},
		{"POST", "application/json", p1Rename, http.StatusOK, nil},
		{"GET", "application/json", &registry.Profile{Username: b6.Username}, http.StatusOK, &env{Data: b6}},
		{"DELETE", "", p1Rename, http.StatusBadRequest, nil},
		{"DELETE", "application/json", nil, http.StatusBadRequest, nil},
		{"DELETE", "application/json", &registry.Profile{Username: p1.Username, ProfileID: p1.ProfileID, Signature: p1.Signature}, http.StatusBadRequest, nil},
		{"DELETE", "application/json", p1Rename, http.StatusOK, nil},
	}

	for i, c := range cases {
		req, err := http.NewRequest(c.method, fmt.Sprintf("%s/registry/profile", s.URL), nil)
		if err != nil {
			t.Errorf("case %d error creating request: %s", i, err.Error())
			continue
		}

		if c.contentType != "" {
			req.Header.Set("Content-Type", c.contentType)
		}
		if c.profile != nil {
			data, err := json.Marshal(c.profile)
			if err != nil {
				t.Errorf("error marshaling json body: %s", err.Error())
				return
			}
			req.Body = ioutil.NopCloser(bytes.NewReader([]byte(data)))
		}

		res, err := http.DefaultClient.Do(req)
		if res.StatusCode != c.resStatus {
			t.Errorf("case %d res status mismatch. expected: %d, got: %d", i, c.resStatus, res.StatusCode)
			continue
		}

		if c.res != nil {
			e := &env{}
			if err := json.NewDecoder(res.Body).Decode(e); err != nil {
				t.Errorf("case %d error reading response body: %s", i, err.Error())
				continue
			}

			// if len(e.Data) != len(c.res.Data) {
			// 	t.Errorf("case %d reponse body mismatch. expected %d, got: %d", i, len(e.Data), len(c.res.Data))
			// 	continue
			// }
			if e.Data.Username != c.res.Data.Username {
				t.Errorf("case %d reponse username mismatch. expected %s, got: %s", i, e.Data.Username, c.res.Data.Username)
			}

			// TODO - check each response for profile matches
		}
	}
}

func TestProfiles(t *testing.T) {
	s := httptest.NewServer(NewRoutes(registry.Registry{Profiles: registry.NewMemProfiles()}))

	p1, err := registry.ProfileFromPrivateKey(&registry.Profile{Username: "b5"}, privKey1)
	if err != nil {
		t.Errorf("error generating profile: %s", err.Error())
		return
	}

	p1Rename, err := registry.ProfileFromPrivateKey(&registry.Profile{Username: "b6"}, privKey1)
	if err != nil {
		t.Errorf("error generating profile: %s", err.Error())
		return
	}

	type env struct {
		Data []*registry.Profile
		Meta struct {
			Code int
		}
	}

	b5 := &registry.Profile{
		ProfileID: "QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt",
		Username:  "b5",
		PublicKey: "CAASpgIwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQC/7Q7fILQ8hc9g07a4HAiDKE4FahzL2eO8OlB1K99Ad4L1zc2dCg+gDVuGwdbOC29IngMA7O3UXijycckOSChgFyW3PafXoBF8Zg9MRBDIBo0lXRhW4TrVytm4Etzp4pQMyTeRYyWR8e2hGXeHArXM1R/A/SjzZUbjJYHhgvEE4OZy7WpcYcW6K3qqBGOU5GDMPuCcJWac2NgXzw6JeNsZuTimfVCJHupqG/dLPMnBOypR22dO7yJIaQ3d0PFLxiDG84X9YupF914RzJlopfdcuipI+6gFAgBw3vi6gbECEzcohjKf/4nqBOEvCDD6SXfl5F/MxoHurbGBYB2CJp+FAgMBAAE=",
	}

	b6 := &registry.Profile{
		ProfileID: "QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt",
		Username:  "b6",
		PublicKey: "CAASpgIwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQC/7Q7fILQ8hc9g07a4HAiDKE4FahzL2eO8OlB1K99Ad4L1zc2dCg+gDVuGwdbOC29IngMA7O3UXijycckOSChgFyW3PafXoBF8Zg9MRBDIBo0lXRhW4TrVytm4Etzp4pQMyTeRYyWR8e2hGXeHArXM1R/A/SjzZUbjJYHhgvEE4OZy7WpcYcW6K3qqBGOU5GDMPuCcJWac2NgXzw6JeNsZuTimfVCJHupqG/dLPMnBOypR22dO7yJIaQ3d0PFLxiDG84X9YupF914RzJlopfdcuipI+6gFAgBw3vi6gbECEzcohjKf/4nqBOEvCDD6SXfl5F/MxoHurbGBYB2CJp+FAgMBAAE=",
	}

	cases := []struct {
		method      string
		endpoint    string
		contentType string
		profile     *registry.Profile
		resStatus   int
		res         *env
	}{
		{"GET", "/registry/profiles", "", nil, http.StatusOK, &env{}},
		{"POST", "/registry/profile", "application/json", p1, http.StatusOK, nil},
		{"GET", "/registry/profiles", "", nil, http.StatusOK, &env{Data: []*registry.Profile{b5}}},
		{"POST", "/registry/profile", "application/json", p1Rename, http.StatusOK, nil},
		{"GET", "/registry/profiles", "", nil, http.StatusOK, &env{Data: []*registry.Profile{b6}}},
		{"DELETE", "/registry/profile", "application/json", p1Rename, http.StatusOK, nil},
		{"GET", "/registry/profiles", "", nil, http.StatusOK, &env{Data: []*registry.Profile{}}},
	}

	for i, c := range cases {
		req, err := http.NewRequest(c.method, fmt.Sprintf("%s%s", s.URL, c.endpoint), nil)
		if err != nil {
			t.Errorf("case %d error creating request: %s", i, err.Error())
			continue
		}

		if c.contentType != "" {
			req.Header.Set("Content-Type", c.contentType)
		}
		if c.profile != nil {
			data, err := json.Marshal(c.profile)
			if err != nil {
				t.Errorf("error marshaling json body: %s", err.Error())
				return
			}
			req.Body = ioutil.NopCloser(bytes.NewReader([]byte(data)))
		}

		res, err := http.DefaultClient.Do(req)
		if res.StatusCode != c.resStatus {
			t.Errorf("case %d res status mismatch. expected: %d, got: %d", i, c.resStatus, res.StatusCode)
			continue
		}

		if c.res != nil {
			e := &env{}
			if err := json.NewDecoder(res.Body).Decode(e); err != nil {
				t.Errorf("case %d error reading response body: %s", i, err.Error())
				continue
			}

			if len(e.Data) != len(c.res.Data) {
				t.Errorf("case %d reponse body mismatch. expected %d, got: %d", i, len(e.Data), len(c.res.Data))
				continue
			}

			// TODO - check each response for profile matches
		}
	}
}

func TestPostProfiles(t *testing.T) {
	un := "username"
	pw := "password"
	s := httptest.NewServer(NewRoutes(registry.Registry{Profiles: registry.NewMemProfiles()}, AddProtector(NewBAProtector(un, pw))))
	const profiles = `[
	{
    "ProfileID": "QmamJUR83rGtDMEvugcC2gtLDx2nhZUTzpzhH6MA2Pb3Md",
    "Handle": "EDGI",
    "PublicKey": "CAASpgIwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQCmTFRx/6dKmoxje8AG+jFv94IcGUGnjrupa7XEr12J/c4ZLn3aPrD8F0tjRbstt1y/J+bO7Qb69DGiu2iSIqyE21nl2oex5+14jtxbupRq9jRTbpUHRj+y9I7uUDwl0E2FS1IQpBBfEGzDPIBVavxbhguC3O3XA7Aq7vea2lpJ1tWpr0GDRYSNmJAybkHS6k7dz1eVXFK+JE8FGFJi/AThQZKWRijvWFdlZvb8RyNFRHzpbr9fh38bRMTqhZpw/YGO5Ly8PNSiOOE4Y5cNUHLEYwG2/lpT4l53iKScsaOazlRkJ6NmkM1il7riCa55fcIAQZDtaAx+CT5ZKfmek4P5AgMBAAE=",
    "Created": "2018-05-01T22:31:18.288004308Z"
  },
  {
    "ProfileID": "QmSyDX5LYTiwQi861F5NAwdHrrnd1iRGsoEvCyzQMUyZ4W",
    "Handle": "b5",
    "PublicKey": "CAASpgIwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQC/W17VPFX+pjyM1MHI1CsvIe+JYQX45MJNITTd7hZZDX2rRWVavGXhsccmVDGU6ubeN3t6ewcBlgCxvyewwKhmZKCAs3/0xNGKXK/YMyZpRVjTWw9yPU9gOzjd9GuNJtL7d1Hl7dPt9oECa7WBCh0W9u2IoHTda4g8B2mK92awLOZTjXeA7vbhKKX+QVHKDxEI0U2/ooLYJUVxEoHRc+DUYNPahX5qRgJ1ZDP4ep1RRRoZR+HjGhwgJP+IwnAnO5cRCWUbZvE1UBJUZDvYMqW3QvDp+TtXwqUWVvt69tp8EnlBgfyXU91A58IEQtLgZ7klOzdSEJDP+S8HIwhG/vbTAgMBAAE=",
    "Created": "2018-04-19T22:10:49.909268968Z"
  }
]`

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/registry/profiles", s.URL), strings.NewReader(profiles))
	if err != nil {
		t.Error(err.Error())
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(un, pw)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Error(err.Error())
	}

	if res.StatusCode != 200 {
		t.Errorf("response status mismatch. expected 200, got: %d", res.StatusCode)
	}

}
