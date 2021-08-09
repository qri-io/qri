// Package dsfs glues datsets to cafs (content-addressed-file-system)
package dsfs

import (
	"fmt"

	golog "github.com/ipfs/go-log"
)

var (
	log = golog.Logger("dsfs")
	// ErrNoChanges indicates a save failed because no values changed, and
	// force-saving was disabled
	ErrNoChanges = fmt.Errorf("no changes")
	// ErrNoReadme is the error for asking a dataset without a readme component
	// for readme info
	ErrNoReadme = fmt.Errorf("this dataset has no readme component")
	// ErrNoTransform is the error for asking a dataset without a tranform
	// component for transform info
	ErrNoTransform = fmt.Errorf("this dataset has no transform component")
	// ErrNoViz is the error for asking a dataset without a viz component for
	// viz info
	ErrNoViz = fmt.Errorf("this dataset has no viz component")
	// ErrStrictMode indicates a dataset failed validation when it is required to
	// pass (Structure.Strict == true)
	ErrStrictMode = fmt.Errorf("dataset body did not validate against schema in strict-mode")
)
