package dsref

import (
	"time"
)

// Info describes a version of a datsaet
type Info struct {
	// Refer
	Ref Ref `json:"ref,omitempty"`

	// FSIPath is this dataset's link to the local filesystem if one exists
	FSIPath string `json:"fsiPath,omitempty"`
	// Published indicates whether this reference is listed as an available dataset
	Published bool `json:"published"`
	// If true, this version doesn't exist locally
	Foreign bool `json:"foreign,omitempty"`

	// Creation timestamp
	Timestamp time.Time `json:"timestamp,omitempty"`
	// Title field from dataset.commit component
	CommitTitle string `json:"commitTitle,omitempty"`
}
