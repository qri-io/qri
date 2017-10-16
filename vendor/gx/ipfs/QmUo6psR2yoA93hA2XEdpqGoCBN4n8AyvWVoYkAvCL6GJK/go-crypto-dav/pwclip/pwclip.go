package pwclip

import (
	"gx/ipfs/QmUo6psR2yoA93hA2XEdpqGoCBN4n8AyvWVoYkAvCL6GJK/go-crypto-dav/drbg"
	"gx/ipfs/Qme1boxspcQWR8FBzMxeppqug2fYgYc15diNWmqgDVnvn2/go-crypto/scrypt"
)

// Password Metadata
type PWM struct {
	URL      string
	Username string
	Extra    *string // Optional
	Prefix   string
	Charset  string // 1 <= utf8.RuneCountInString(Charset) <= 256
	Length   int    // Length in runes (Unicode code points)
}

const CharsetAlphaNumeric = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

func (pwm *PWM) Password(key []byte) string {
	rng := drbg.New(key)
	rng.Reseed([]byte(pwm.URL))
	rng.Reseed([]byte(pwm.Username))
	if pwm.Extra != nil {
		rng.Reseed([]byte(*pwm.Extra))
	}

	charset := []rune(pwm.Charset)
	m := 256 % len(pwm.Charset)
	pw := []rune(pwm.Prefix)
	buf := make([]byte, 256)

	for len(pw) < pwm.Length {
		rng.Read(buf)
		for i := 0; i < len(buf) && len(pw) < pwm.Length; i++ {
			r := int(buf[i])
			// ensure uniform distribution mod len(charset)
			if r < 256-m {
				pw = append(pw, charset[r%len(charset)])
			}
		}
	}

	return string(pw[:pwm.Length])
}

func Key(passphrase []byte) ([]byte, error) {
	return scrypt.Key(passphrase, []byte("pwclip"), 2<<15, 8, 1, 32)
}
