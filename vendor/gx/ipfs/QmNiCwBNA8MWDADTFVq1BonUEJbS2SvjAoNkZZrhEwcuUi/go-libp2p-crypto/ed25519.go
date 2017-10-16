package crypto

import (
	"bytes"
	"fmt"
	"io"

	pb "gx/ipfs/QmNiCwBNA8MWDADTFVq1BonUEJbS2SvjAoNkZZrhEwcuUi/go-libp2p-crypto/pb"
	"gx/ipfs/QmQ51pHe6u7CWodkUGDLqaCEMchkbMt7VEZnECF5mp6tVb/ed25519"
	proto "gx/ipfs/QmZ4Qi3GaRbjcx28Sme5eMH7RQjGkt8wHxt2a65oLaeFEV/gogo-protobuf/proto"
)

type Ed25519PrivateKey struct {
	sk *[64]byte
	pk *[32]byte
}

type Ed25519PublicKey struct {
	k *[32]byte
}

func GenerateEd25519Key(src io.Reader) (PrivKey, PubKey, error) {
	pub, priv, err := ed25519.GenerateKey(src)
	if err != nil {
		return nil, nil, err
	}

	return &Ed25519PrivateKey{
			sk: priv,
			pk: pub,
		},
		&Ed25519PublicKey{
			k: pub,
		},
		nil
}

func (k *Ed25519PrivateKey) Bytes() ([]byte, error) {
	pbmes := new(pb.PrivateKey)
	typ := pb.KeyType_Ed25519
	pbmes.Type = &typ

	buf := make([]byte, 96)
	copy(buf, k.sk[:])
	copy(buf[64:], k.pk[:])
	pbmes.Data = buf
	return proto.Marshal(pbmes)
}

func (k *Ed25519PrivateKey) Equals(o Key) bool {
	edk, ok := o.(*Ed25519PrivateKey)
	if !ok {
		return false
	}

	return bytes.Equal((*k.sk)[:], (*edk.sk)[:]) && bytes.Equal((*k.pk)[:], (*edk.pk)[:])
}

func (k *Ed25519PrivateKey) GetPublic() PubKey {
	return &Ed25519PublicKey{k.pk}
}

func (k *Ed25519PrivateKey) Hash() ([]byte, error) {
	return KeyHash(k)
}

func (k *Ed25519PrivateKey) Sign(msg []byte) ([]byte, error) {
	out := ed25519.Sign(k.sk, msg)
	return (*out)[:], nil
}

func (k *Ed25519PublicKey) Bytes() ([]byte, error) {
	pbmes := new(pb.PublicKey)
	typ := pb.KeyType_Ed25519
	pbmes.Type = &typ
	pbmes.Data = (*k.k)[:]
	return proto.Marshal(pbmes)
}

func (k *Ed25519PublicKey) Equals(o Key) bool {
	edk, ok := o.(*Ed25519PublicKey)
	if !ok {
		return false
	}

	return bytes.Equal((*k.k)[:], (*edk.k)[:])
}

func (k *Ed25519PublicKey) Hash() ([]byte, error) {
	return KeyHash(k)
}

func (k *Ed25519PublicKey) Verify(data []byte, sig []byte) (bool, error) {
	var asig [64]byte
	copy(asig[:], sig)
	return ed25519.Verify(k.k, data, &asig), nil
}

func UnmarshalEd25519PrivateKey(data []byte) (*Ed25519PrivateKey, error) {
	if len(data) != 96 {
		return nil, fmt.Errorf("expected ed25519 data size to be 96")
	}
	var priv [64]byte
	var pub [32]byte
	copy(priv[:], data)
	copy(pub[:], data[64:])

	return &Ed25519PrivateKey{
		sk: &priv,
		pk: &pub,
	}, nil
}
