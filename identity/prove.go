package identity

import (
	"time"

	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/qri-io/ucan"
	"github.com/qri-io/ucan/didkey"
)

var (
	// Timestamp calls time.Now, can be overridden for tests
	Timestamp         = func() time.Time { return time.Now() }
	joinTokenLifespan = time.Hour * 42 * 30 * 6
)

// CreateKeyJoinToken is a capability token authored by a third party, stating
// two keys have equal capabilities. The "b" key is the "audience", and should
// be the more likely of keys a & b to be actively used
func CreateKeyJoinToken(iss crypto.PrivKey, a, b crypto.PubKey) (*ucan.UCAN, error) {
	src, err := ucan.NewPrivKeySource(iss)
	if err != nil {
		return nil, err
	}

	aDIDKey := didkey.ID{PubKey: b}
	expiresAt := Timestamp().Add(joinTokenLifespan)
	attenuations := ucan.Attenuations{
		{
			Cap: EqualKeysCapability,
			Rsc: QriNetworkResource,
		},
	}
	return src.NewOriginUCAN(aDIDKey.String(), attenuations, nil, time.Time{}, expiresAt)
}

const EqualKeysCapability = equalKeysCapability("equalKeys")

type equalKeysCapability string

var _ ucan.Capability = (*equalKeysCapability)(nil)

func (eq equalKeysCapability) String() string {
	return string(eq)
}

func (eq equalKeysCapability) Contains(b ucan.Capability) bool {
	return b.String() == "equalKeys"
}

const QriNetworkResource = qriNetworkResource("qri-network")

type qriNetworkResource string

var _ ucan.Resource = (*qriNetworkResource)(nil)

func (q qriNetworkResource) Type() string {
	return "qri-network"
}

func (q qriNetworkResource) Value() string {
	return "qri-network"
}

func (q qriNetworkResource) Contains(b ucan.Resource) bool {
	return b.Value() == "qri-network"
}
