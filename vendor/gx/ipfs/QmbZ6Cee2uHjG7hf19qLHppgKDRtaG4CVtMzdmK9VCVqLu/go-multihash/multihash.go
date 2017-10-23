package multihash

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"math"

	b58 "gx/ipfs/QmT8rehPR3F6bmwL6zjUN8XpiDBFFpMP2myPdC6ApsWfJf/go-base58"
)

// errors
var (
	ErrUnknownCode      = errors.New("unknown multihash code")
	ErrTooShort         = errors.New("multihash too short. must be > 3 bytes")
	ErrTooLong          = errors.New("multihash too long. must be < 129 bytes")
	ErrLenNotSupported  = errors.New("multihash does not yet support digests longer than 127 bytes")
	ErrInvalidMultihash = errors.New("input isn't valid multihash")

	ErrVarintBufferShort = errors.New("uvarint: buffer too small")
	ErrVarintTooLong     = errors.New("uvarint: varint too big (max 64bit)")
)

// ErrInconsistentLen is returned when a decoded multihash has an inconsistent length
type ErrInconsistentLen struct {
	dm *DecodedMultihash
}

func (e ErrInconsistentLen) Error() string {
	return fmt.Sprintf("multihash length inconsistent: %v", e.dm)
}

// constants
const (
	SHA1     = 0x11
	SHA2_256 = 0x12
	SHA2_512 = 0x13
	SHA3     = 0x14

	BLAKE2B_MIN = 0xb201
	BLAKE2B_MAX = 0xb240
	BLAKE2S_MIN = 0xb241
	BLAKE2S_MAX = 0xb260

	DBL_SHA2_256 = 0x56
)

func init() {
	// Add blake2b (64 codes)
	for c := uint64(BLAKE2B_MIN); c <= BLAKE2B_MAX; c++ {
		n := c - BLAKE2B_MIN + 1
		name := fmt.Sprintf("blake2b-%d", n*8)
		Names[name] = c
		Codes[c] = name
		DefaultLengths[c] = int(n)
	}

	// Add blake2s (32 codes)
	for c := uint64(BLAKE2S_MIN); c <= BLAKE2S_MAX; c++ {
		n := c - BLAKE2S_MIN + 1
		name := fmt.Sprintf("blake2s-%d", n*8)
		Names[name] = c
		Codes[c] = name
		DefaultLengths[c] = int(n)
	}
}

// Names maps the name of a hash to the code
var Names = map[string]uint64{
	"sha1":         SHA1,
	"sha2-256":     SHA2_256,
	"sha2-512":     SHA2_512,
	"sha3":         SHA3,
	"dbl-sha2-256": DBL_SHA2_256,
}

// Codes maps a hash code to it's name
var Codes = map[uint64]string{
	SHA1:         "sha1",
	SHA2_256:     "sha2-256",
	SHA2_512:     "sha2-512",
	SHA3:         "sha3",
	DBL_SHA2_256: "dbl-sha2-256",
}

// DefaultLengths maps a hash code to it's default length
var DefaultLengths = map[uint64]int{
	SHA1:         20,
	SHA2_256:     32,
	SHA2_512:     64,
	SHA3:         64,
	DBL_SHA2_256: 32,
}

func uvarint(buf []byte) (uint64, []byte, error) {
	n, c := binary.Uvarint(buf)

	if c == 0 {
		return n, buf, ErrVarintBufferShort
	} else if c < 0 {
		return n, buf[-c:], ErrVarintTooLong
	} else {
		return n, buf[c:], nil
	}
}

type DecodedMultihash struct {
	Code   uint64
	Name   string
	Length int // Length is just int as it is type of len() opearator
	Digest []byte
}

type Multihash []byte

func (m *Multihash) HexString() string {
	return hex.EncodeToString([]byte(*m))
}

func (m *Multihash) String() string {
	return m.HexString()
}

func FromHexString(s string) (Multihash, error) {
	b, err := hex.DecodeString(s)
	if err != nil {
		return Multihash{}, err
	}

	return Cast(b)
}

func (m Multihash) B58String() string {
	return b58.Encode([]byte(m))
}

func FromB58String(s string) (m Multihash, err error) {
	// panic handler, in case we try accessing bytes incorrectly.
	defer func() {
		if e := recover(); e != nil {
			m = Multihash{}
			err = e.(error)
		}
	}()

	//b58 smells like it can panic...
	b := b58.Decode(s)
	if len(b) == 0 {
		return Multihash{}, ErrInvalidMultihash
	}

	return Cast(b)
}

func Cast(buf []byte) (Multihash, error) {
	dm, err := Decode(buf)
	if err != nil {
		return Multihash{}, err
	}

	if !ValidCode(dm.Code) {
		return Multihash{}, ErrUnknownCode
	}

	return Multihash(buf), nil
}

// Decode a hash from the given Multihash.
func Decode(buf []byte) (*DecodedMultihash, error) {

	if len(buf) < 3 {
		return nil, ErrTooShort
	}

	var err error
	var code, length uint64

	code, buf, err = uvarint(buf)
	if err != nil {
		return nil, err
	}

	length, buf, err = uvarint(buf)
	if err != nil {
		return nil, err
	}

	if length > math.MaxInt32 {
		return nil, errors.New("digest too long, supporting only <= 2^31-1")
	}

	dm := &DecodedMultihash{
		Code:   code,
		Name:   Codes[code],
		Length: int(length),
		Digest: buf,
	}

	if len(dm.Digest) != dm.Length {
		return nil, ErrInconsistentLen{dm}
	}

	return dm, nil
}

// Encode a hash digest along with the specified function code.
// Note: the length is derived from the length of the digest itself.
func Encode(buf []byte, code uint64) ([]byte, error) {

	if !ValidCode(code) {
		return nil, ErrUnknownCode
	}

	start := make([]byte, 2*binary.MaxVarintLen64)
	spot := start
	n := binary.PutUvarint(spot, code)
	spot = start[n:]
	n += binary.PutUvarint(spot, uint64(len(buf)))

	return append(start[:n], buf...), nil
}

func EncodeName(buf []byte, name string) ([]byte, error) {
	return Encode(buf, Names[name])
}

// ValidCode checks whether a multihash code is valid.
func ValidCode(code uint64) bool {
	if AppCode(code) {
		return true
	}

	if _, ok := Codes[code]; ok {
		return true
	}

	return false
}

// AppCode checks whether a multihash code is part of the App range.
func AppCode(code uint64) bool {
	return code >= 0 && code < 0x10
}
