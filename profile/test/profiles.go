package test

import (
	"fmt"

	testkeys "github.com/qri-io/qri/auth/key/test"
	"github.com/qri-io/qri/profile"
)

var profiles map[string]*profile.Profile

func init() {
	kd1 := testkeys.GetKeyData(0)
	kd2 := testkeys.GetKeyData(1)
	kd9 := testkeys.GetKeyData(9)

	profiles = map[string]*profile.Profile{
		"kermit":          {Peername: "kermit", Email: "kermit+test_user@qri.io", ID: profile.IDB58MustDecode(kd1.EncodedPeerID), PrivKey: kd1.PrivKey, PubKey: kd1.PrivKey.GetPublic()},
		"miss_piggy":      {Peername: "miss_piggy", Email: "miss_piggy+test_user@qri.io", ID: profile.IDB58MustDecode(kd2.EncodedPeerID), PrivKey: kd2.PrivKey, PubKey: kd2.PrivKey.GetPublic()},
		"yolanda_the_rat": {Peername: "yolanda_the_rat", Email: "yolanda+test_user@qri.io", ID: profile.IDB58MustDecode(kd9.EncodedPeerID), PrivKey: kd9.PrivKey, PubKey: kd9.PrivKey.GetPublic()},
	}
}

// GetProfile fetches a profile for testing
func GetProfile(username string) *profile.Profile {
	pro, ok := profiles[username]
	if !ok {
		panic(fmt.Sprintf("test username %q does not exits", username))
	}
	return pro
}
