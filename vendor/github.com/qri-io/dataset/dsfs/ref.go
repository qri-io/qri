package dsfs

import (
	"strings"

	"github.com/ipfs/go-datastore"
)

// RefType examines input, trying to determine weather the reference in question
// is a name or a path. If a path is suspected, it attempts to cannonicalize the
// reference
func RefType(input string) (refType, ref string) {
	if strings.HasPrefix(input, "/ipfs/") || strings.HasSuffix(input, PackageFileDataset.Filename()) {
		return "path", cleanHash(input)
	} else if len(input) == 46 && !strings.Contains(input, "_") {
		// 46 is the current length of a base58-encoded hash on ipfs
		return "path", cleanHash(input)
	}
	return "name", input
}

// cleanHash returns a canonicalized hash reference with network path prefix
func cleanHash(in string) string {
	if !strings.HasPrefix(in, "/ipfs/") {
		in = datastore.NewKey("/ipfs/").ChildString(in).String()
	}
	if !strings.HasSuffix(in, PackageFileDataset.String()) {
		in = datastore.NewKey(in).ChildString(PackageFileDataset.String()).String()
	}
	return in
}
