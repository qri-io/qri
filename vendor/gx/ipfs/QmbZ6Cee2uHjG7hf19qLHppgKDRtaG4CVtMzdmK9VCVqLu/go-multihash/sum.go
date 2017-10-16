package multihash

import (
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"errors"
	"fmt"

	blake2b "gx/ipfs/QmaPHkZLbQQbvcyavn8q1GFHg6o6yeceyHFSJ3Pjf3p3TQ/go-crypto/blake2b"
	blake2s "gx/ipfs/QmaPHkZLbQQbvcyavn8q1GFHg6o6yeceyHFSJ3Pjf3p3TQ/go-crypto/blake2s"
	sha3 "gx/ipfs/QmaPHkZLbQQbvcyavn8q1GFHg6o6yeceyHFSJ3Pjf3p3TQ/go-crypto/sha3"
)

var ErrSumNotSupported = errors.New("Function not implemented. Complain to lib maintainer.")

func Sum(data []byte, code uint64, length int) (Multihash, error) {
	m := Multihash{}
	err := error(nil)
	if !ValidCode(code) {
		return m, fmt.Errorf("invalid multihash code %d", code)
	}

	if length < 0 {
		var ok bool
		length, ok = DefaultLengths[code]
		if !ok {
			return m, fmt.Errorf("no default length for code %d", code)
		}
	}

	var d []byte
	switch {
	case isBlake2s(code):
		olen := code - BLAKE2S_MIN + 1
		switch olen {
		case 32:
			out := blake2s.Sum256(data)
			d = out[:]
		default:
			return nil, fmt.Errorf("unsupported length for blake2s: %d", olen)
		}
	case isBlake2b(code):
		olen := code - BLAKE2B_MIN + 1
		switch olen {
		case 32:
			out := blake2b.Sum256(data)
			d = out[:]
		case 48:
			out := blake2b.Sum384(data)
			d = out[:]
		case 64:
			out := blake2b.Sum512(data)
			d = out[:]
		default:
			return nil, fmt.Errorf("unsupported length for blake2b: %d", olen)
		}
	default:
		switch code {
		case SHA1:
			d = sumSHA1(data)
		case SHA2_256:
			d = sumSHA256(data)
		case SHA2_512:
			d = sumSHA512(data)
		case SHA3:
			d, err = sumSHA3(data)
		case DBL_SHA2_256:
			d = sumSHA256(sumSHA256(data))
		default:
			return m, ErrSumNotSupported
		}
	}
	if err != nil {
		return m, err
	}
	return Encode(d[0:length], code)
}

func isBlake2s(code uint64) bool {
	return code >= BLAKE2S_MIN && code <= BLAKE2S_MAX
}
func isBlake2b(code uint64) bool {
	return code >= BLAKE2B_MIN && code <= BLAKE2B_MAX
}

func sumSHA1(data []byte) []byte {
	a := sha1.Sum(data)
	return a[0:20]
}

func sumSHA256(data []byte) []byte {
	a := sha256.Sum256(data)
	return a[0:32]
}

func sumSHA512(data []byte) []byte {
	a := sha512.Sum512(data)
	return a[0:64]
}

func sumSHA3(data []byte) ([]byte, error) {
	h := sha3.New512()
	if _, err := h.Write(data); err != nil {
		return nil, err
	}
	return h.Sum(nil), nil
}
