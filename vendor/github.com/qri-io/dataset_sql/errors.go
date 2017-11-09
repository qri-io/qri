package dataset_sql

import (
	"fmt"
)

// NotYetImplemented reports missing features. it'd be lovely to not need this ;)
func NotYetImplemented(feature string) error {
	return fmt.Errorf("%s are not implemented. check docs.qri.io/releases for info", feature)
}

func ErrUnrecognizedReference(ref string) error {
	return fmt.Errorf("unrecognized reference: %s", ref)
}

func ErrAmbiguousReference(ref string) error {
	return fmt.Errorf("reference: %s is ambiguous", ref)
}
