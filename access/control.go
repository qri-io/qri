package access

import (
	"fmt"

	"github.com/ipfs/go-cid"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/key"
	"github.com/qri-io/ucan"
)

type Control interface {
	Grant(subjectID, objectID cid.ID, cap ucan.Capability) (*ucan.Token, error)
	Revoke(tokenID cid.ID) error
	TokensForObject(objectID string) ([]*ucan.Token, error)
}

func NewControl(fs qfs.Filesystem, ks key.Store) (Control, error) {
	return control{
		ks: ks,
		ts: ucan.NewMemTokenStore(),
	}, nil
}

type control struct {
	ks     key.Store
	ts     ucan.TokenStore
	parser *ucan.TokenParser
}

var _ Control = (*control)(nil)

func (c control) Grant(subjectId, objectID string, cap ucan.Capability) (*ucan.Token, error) {

	return nil, fmt.Errorf("not finished")
}

func (c control) RevokeToken(tokenID string) error {
	return fmt.Errorf("not finished")
}

func (c control) TokensForObject(id string) ([]*ucan.Token, error) {

}

func (c control) newParser(store ucan.TokenStore, cidResolver ucan.CIDBytesResolver) *ucan.TokenParser {
	return ucan.NewTokenParser(parseAttenuations, ucan.StringDIDPubKeyResolver{}, store.(ucan.CIDBytesResolver))
}

var datasetCapabilities = ucan.NewNestedCapabilities("SUPER_USER", "OVERWRITE", "SOFT_DELETE", "REVISE", "CREATE")

func parseAttenuations(m map[string]interface{}) (ucan.Attenuation, error) {
	var (
		cap string
		rsc ucan.Resource
	)

	for key, vali := range m {
		val, ok := vali.(string)
		if !ok {
			return ucan.Attenuation{}, fmt.Errorf(`expected attenuation value to be a string`)
		}

		if key == ucan.CapKey {
			cap = val
		} else {
			rsc = ucan.NewStringLengthResource(key, val)
		}
	}

	return ucan.Attenuation{
		Rsc: rsc,
		Cap: datasetCapabilities.Cap(cap),
	}, nil
}
