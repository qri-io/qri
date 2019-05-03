// Qri is a distributed dataset version control tool. Bigger than a spreadsheet,
// smaller than a database, datasets are all around us.
// Use Qri to browse, download, create, fork, and publish datasets on a peer-to-peer
// network that works both on and offline.
//
// more info at: https://qri.io
package main

import (
	// "github.com/pkg/profile"
	"github.com/qri-io/qri/cmd"
)

func main() {
	// defer profile.Start(profile.MemProfile).Stop()
	cmd.Execute()
}
