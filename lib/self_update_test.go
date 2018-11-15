package lib

import (
	"context"
	"testing"

	namesys "gx/ipfs/QmUJYo4etAQqFfSS2rarFAE97eNGB8ej64YkRT2SmsYD4r/go-ipfs/namesys"
)

func TestCheckVersion(t *testing.T) {
	name := PrevIPNSName
	ver := LastPubVerHash
	defer func() {
		PrevIPNSName = name
		LastPubVerHash = ver
	}()

	res := namesys.NewDNSResolver()

	PrevIPNSName = "foo"
	expect := "error resolving name: not a valid domain name"
	if _, err := CheckVersion(context.Background(), res, name, ver); err != nil && err.Error() != expect {
		t.Errorf("error mismatch. epxected: '%s', got: '%s'", expect, err.Error())
		return
	}

	// TODO - not workin'
	// if err := CheckVersion(context.Background(), res); err != nil {
	// 	t.Errorf("error checking valid version: %s", err.Error())
	// 	return
	// }

	// lastPubVerHash = "/def/not/good"
	// if err := CheckVersion(context.Background(), res); err != nil && err != ErrUpdateRequired {
	// 	t.Errorf("expected ErrUpdateRequired, got: %s", err.Error())
	// } else if err == nil {
	// 	t.Errorf("expected error, got nil")
	// }
}
