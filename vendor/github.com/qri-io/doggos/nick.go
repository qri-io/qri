// doggos is a v important package for make usernames on qri.
package doggos

import (
	"fmt"

	"github.com/jbenet/go-base58"
)

// DoggoNick returns a unique-ish username from a base58-encoded string
// doggo nick's are a color concatenated with a dog breed, where sum
// of the first half determines the color, and sum of the second determines
// the breed.
func DoggoNick(base58id string) string {
	spl := len(base58id) / 2
	colorSum := base58Sum(base58id[:spl])
	color := colors[colorSum%len(colors)]
	doggoSum := base58Sum(base58id[spl:])
	doggo := doggos[doggoSum%len(doggos)]

	return fmt.Sprintf("%s_%s", color, doggo)
}

func base58Sum(in string) (sum int) {
	for _, ch := range in {
		for i, ach := range base58.BTCAlphabet {
			if ch == ach {
				sum += i
				break
			}
		}
	}
	return
}
