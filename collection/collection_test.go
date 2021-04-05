package collection_test

import (
	"testing"

	"github.com/qri-io/qri/collection"
	"github.com/qri-io/qri/collection/spec"
)

func TestLocalCollection(t *testing.T) {
	spec.AssertCollectionSpec(t, collection.NewLocalCollection)
}
