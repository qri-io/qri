package cmd

import (
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/lib"
)

// EnsureFSIAgrees should be passed to GetCurrentRefSelect in order to ensure that any references
// used by a command have agreement between what their .qri-ref linkfile thinks and what the
// qri repository thinks. If there's a disagreement, the linkfile wins and the repository will
// be updated to match.
// This is useful if a user has a working directory, and then manually deletes the .qri-ref (which
// will unlink the dataset), or renames / moves the directory and then runs a command in that
// directory (which will update the repository with the new working directory's path).
func EnsureFSIAgrees(f *lib.FSIMethods) *FSIRefLinkEnsurer {
	if f == nil {
		return nil
	}
	return &FSIRefLinkEnsurer{FSIMethods: f}
}

// FSIRefLinkEnsurer is a simple wrapper for ensuring the linkfile agrees with the repository. We
// use it instead of a raw FSIMethods pointer so that users of this code see they need to call
// EnsureFSIAgrees(*fsiMethods) when calling GetRefSelect, hopefully providing a bit of insight
// about what this parameter is for.
type FSIRefLinkEnsurer struct {
	FSIMethods *lib.FSIMethods
}

// EnsureRef checks if the linkfile and repository agree on the dataset's working directory path.
// If not, it will modify the repository so that it matches the linkfile. The linkfile will
// never be modified.
func (e *FSIRefLinkEnsurer) EnsureRef(refs *RefSelect) error {
	if e == nil {
		return nil
	}
	p := lib.EnsureParams{Dir: refs.Dir(), Ref: refs.Ref()}
	info := dsref.VersionInfo{}
	// Lib call matches the gorpc method signature, but `out` is not used
	err := e.FSIMethods.EnsureRef(&p, &info)
	return err
}
