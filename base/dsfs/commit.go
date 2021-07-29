package dsfs

import (
	"context"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"time"

	crypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsio"
	"github.com/qri-io/deepdiff"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/base/friendly"
	"github.com/qri-io/qri/base/toqtype"
	"github.com/qri-io/qri/event"
)

// Timestamp is an function for getting commit timestamps
// timestamps MUST be stored in UTC time zone
var Timestamp = func() time.Time {
	return time.Now().UTC()
}

// BodyAction represents the action that should be taken to understand how the
// body changed
type BodyAction string

const (
	// BodyDefault is the default action: compare them to get how much changed
	BodyDefault BodyAction = "default"
	// BodySame means that the bodies are the same, no need to compare
	BodySame BodyAction = "same"
	// BodyTooBig means the body is too big to directly compare, and should use
	// some other method
	BodyTooBig BodyAction = "too_big"
)

func commitFileAddFunc(ctx context.Context, privKey crypto.PrivKey, publisher event.Publisher) writeComponentFunc {
	return func(src qfs.Filesystem, dst qfs.MerkleDagStore, prev, ds *dataset.Dataset, added qfs.Links, sw *SaveSwitches) error {
		if ds.Commit == nil {
			return errNoComponent
		}

		if evtErr := publisher.Publish(ctx, event.ETDatasetSaveProgress, event.DsSaveEvent{
			Username:   ds.Peername,
			Name:       ds.Name,
			Message:    "finalizing",
			Completion: 0.9,
		}); evtErr != nil {
			log.Debugw("publish event errored", "error", evtErr)
		}

		log.Debugw("writing commit file", "bodyAction", sw.bodyAct, "force", sw.ForceIfNoChanges, "fileHint", sw.FileHint)

		updateScriptPaths(dst, ds, added)

		if err := confirmByteChangesExist(ds, prev, sw.ForceIfNoChanges, dst, added); err != nil {
			return fmt.Errorf("saving: %w", err)
		}

		if err := ensureCommitTitleAndMessage(ctx, src, privKey, ds, prev, sw.bodyAct, sw.FileHint, sw.ForceIfNoChanges); err != nil {
			log.Debugf("ensureCommitTitleAndMessage: %s", err)
			return fmt.Errorf("saving: %w", err)
		}

		ds.DropTransientValues()
		setComponentRefs(dst, ds, bodyFilename(ds), added)

		signedBytes, err := privKey.Sign(ds.SigningBytes())
		if err != nil {
			log.Debug(err.Error())
			return fmt.Errorf("signing commit: %w", err)
		}
		ds.Commit.Signature = base64.StdEncoding.EncodeToString(signedBytes)
		log.Debugw("writing commit", "title", ds.Commit.Title, "message", ds.Commit.Message)

		f, err := JSONFile(PackageFileCommit.String(), ds.Commit)
		if err != nil {
			return err
		}
		return writePackageFile(dst, f, added)
	}
}

// confirmByteChangesExist returns an early error if no components paths
// differ from the previous flag & we're not forcing a commit.
// if we are forcing a commit, set commit title and message values, which
// triggers a fast-path in ensureCommitTitleAndMessage
//
// keep in mind: it is possible for byte-level changes to exist, but not cause
// any alterations to dataset values, (for example: removing non-sensitive
// whitespace)
func confirmByteChangesExist(ds, prev *dataset.Dataset, force bool, dst qfs.MerkleDagStore, added qfs.Links) error {
	if force {
		log.Debugf("forcing changes. skipping uniqueness checks")
		// fast path: forced changes ignore all comparison
		if ds.Commit.Title == "" {
			ds.Commit.Title = "forced update"
		}
		if ds.Commit.Message == "" {
			ds.Commit.Message = "forced update"
		}
		return nil
	}

	if prev == nil {
		return nil
	}

	// Viz, Readme and Transform components are inlined in the dataset, so they
	// don't have path values before the commit component is finalized.
	// use field equality checks instead of path comparison
	if !ds.Viz.ShallowCompare(prev.Viz) {
		log.Debugf("byte changes exist. viz components are inequal")
		return nil
	}
	if !ds.Readme.ShallowCompare(prev.Readme) {
		log.Debugf("byte changes exist. readme components are inequal")
		return nil
	}
	if !ds.Transform.ShallowCompare(prev.Transform) {
		log.Debugf("byte changes exist. transform components are inequal")
		return nil
	}

	// create path map for previous, ignoring dataset & commit components which
	// don't yet have paths on the next version
	prevRefs := prev.PathMap("dataset", "commit")

	// create an empty dataset & populate it with path references to avoid
	// altering the in-flight dataset
	nextDs := &dataset.Dataset{}
	setComponentRefs(dst, nextDs, bodyFilename(ds), added)
	nextRefs := nextDs.PathMap()

	for key, nextPath := range nextRefs {
		if prevRefs[key] != nextPath {
			log.Debugf("byte changes exist. %q components are inequal", key)
			return nil
		}
	}
	// need to check previous paths in case next version is dropping components
	for key, prevPath := range prevRefs {
		if nextRefs[key] != prevPath {
			log.Debugf("byte changes exist. %q components are inequal", key)
			return nil
		}
	}

	log.Debugw("confirmByteChanges", "err", ErrNoChanges)
	return ErrNoChanges
}

// ensureCommitTitleAndMessage creates the commit and title, message, skipping
// if both title and message are set. If no values are provided a commit
// description is generated by examining changes between the two versions
func ensureCommitTitleAndMessage(ctx context.Context, fs qfs.Filesystem, privKey crypto.PrivKey, ds, prev *dataset.Dataset, bodyAct BodyAction, fileHint string, forceIfNoChanges bool) error {
	if ds.Commit.Title != "" && ds.Commit.Message != "" {
		log.Debugf("commit meta & title are set. skipping commit description calculation")
		return nil
	}

	// fast path when commit and title are set
	log.Debugw("ensureCommitTitleAndMessage", "bodyAct", bodyAct)
	shortTitle, longMessage, err := generateCommitDescriptions(ctx, fs, ds, prev, bodyAct, forceIfNoChanges)
	if err != nil {
		log.Debugf("generateCommitDescriptions err: %s", err)
		return err
	}

	if shortTitle == defaultCreatedDescription && fileHint != "" {
		shortTitle = shortTitle + " from " + filepath.Base(fileHint)
	}
	if longMessage == defaultCreatedDescription && fileHint != "" {
		longMessage = longMessage + " from " + filepath.Base(fileHint)
	}

	if ds.Commit.Title == "" {
		ds.Commit.Title = shortTitle
	}
	if ds.Commit.Message == "" {
		ds.Commit.Message = longMessage
	}

	return nil
}

const defaultCreatedDescription = "created dataset"

// returns a commit message based on the diff of the two datasets
func generateCommitDescriptions(ctx context.Context, fs qfs.Filesystem, ds, prev *dataset.Dataset, bodyAct BodyAction, forceIfNoChanges bool) (short, long string, err error) {
	if prev == nil || prev.IsEmpty() {
		return defaultCreatedDescription, defaultCreatedDescription, nil
	}

	// Inline body if it is a reasonable size, to get message about how the body has changed.
	if bodyAct != BodySame {
		// If previous version had bodyfile, read it and assign it
		if prev.Structure != nil && prev.Structure.Length < BodySizeSmallEnoughToDiff {
			if prev.BodyFile() != nil {
				log.Debugf("inlining body file to calculate a diff")
				if prevReader, err := dsio.NewEntryReader(prev.Structure, prev.BodyFile()); err == nil {
					if prevBodyData, err := dsio.ReadAll(prevReader); err == nil {
						prev.Body = prevBodyData
					}
				}
			}
		}
	}

	// Read the transform files to see if they changed.
	// TODO(dustmop): Would be better to get a line-by-line diff
	if prev.Transform != nil && prev.Transform.ScriptPath != "" {
		log.Debugf("inlining prev transform ScriptPath=%q", prev.Transform.ScriptPath)
		err := prev.Transform.OpenScriptFile(ctx, fs)
		if err != nil {
			log.Error("prev.Transform.ScriptPath %q open err: %s", prev.Transform.ScriptPath, err)
		} else {
			tfFile := prev.Transform.ScriptFile()
			content, err := ioutil.ReadAll(tfFile)
			if err != nil {
				log.Error("prev.Transform.ScriptPath %q read err: %s", prev.Transform.ScriptPath, err)
			}
			prev.Transform.Text = string(content)
		}
	}
	if ds.Transform != nil && ds.Transform.ScriptPath != "" {
		log.Debugf("inlining next transform ScriptPath=%q", ds.Transform.ScriptPath)
		err = ds.Transform.OpenScriptFile(ctx, fs)
		if err != nil {
			log.Errorf("ds.Transform.ScriptPath %q open err: %s", ds.Transform.ScriptPath, err)
		} else {
			tfFile := ds.Transform.ScriptFile()
			content, err := ioutil.ReadAll(tfFile)
			if err != nil {
				log.Errorf("ds.Transform.ScriptPath %q read err: %s", ds.Transform.ScriptPath, err)
			}
			ds.Transform.Text = string(content)
		}
		// Reopen the transform file so that WriteDataset will be able to write it to the store.
		if reopenErr := ds.Transform.OpenScriptFile(ctx, fs); reopenErr != nil {
			log.Debugf("error reopening transform script file: %q", reopenErr)
		}
	}

	// Read the readme files to see if they changed.
	// TODO(dustmop): Would be better to get a line-by-line diff
	if prev.Readme != nil && prev.Readme.ScriptPath != "" {
		log.Debugf("inlining prev readme ScriptPath=%q", prev.Readme.ScriptPath)
		err := prev.Readme.OpenScriptFile(ctx, fs)
		if err != nil {
			log.Error("prev.Readme.ScriptPath %q open err: %s", prev.Readme.ScriptPath, err)
		} else {
			tfFile := prev.Readme.ScriptFile()
			content, err := ioutil.ReadAll(tfFile)
			if err != nil {
				log.Error("prev.Readme.ScriptPath %q read err: %s", prev.Readme.ScriptPath, err)
			}
			prev.Readme.Text = string(content)
		}
	}
	if ds.Readme != nil && ds.Readme.ScriptPath != "" {
		log.Debugf("inlining next readme ScriptPath=%q", ds.Readme.ScriptPath)
		err = ds.Readme.OpenScriptFile(ctx, fs)
		if err != nil {
			log.Debugf("ds.Readme.ScriptPath %q open err: %s", ds.Readme.ScriptPath, err)
			err = nil
		} else {
			tfFile := ds.Readme.ScriptFile()
			content, err := ioutil.ReadAll(tfFile)
			if err != nil {
				log.Errorf("ds.Readme.ScriptPath %q read err: %s", ds.Readme.ScriptPath, err)
			}
			ds.Readme.Text = string(content)
		}
		if reopenErr := ds.Readme.OpenScriptFile(ctx, fs); reopenErr != nil {
			log.Debugf("error reopening readme script file: %q", reopenErr)
		}
	}

	var prevData map[string]interface{}
	prevData, err = toqtype.StructToMap(prev)
	if err != nil {
		return "", "", err
	}

	var nextData map[string]interface{}
	nextData, err = toqtype.StructToMap(ds)
	if err != nil {
		return "", "", err
	}

	// TODO(dustmop): All of this should be using fill and/or component. Would be awesome to
	// be able to do:
	//   prevBody = fill.GetPathValue(prevData, "body")
	//   fill.DeletePathValue(prevData, "body")
	//   component.DropDerivedValues(prevData, "structure")
	var prevBody interface{}
	var nextBody interface{}
	if bodyAct != BodySame {
		prevBody = prevData["body"]
		nextBody = nextData["body"]
	}
	delete(prevData, "body")
	delete(nextData, "body")

	if prevTransform, ok := prevData["transform"]; ok {
		if prevObject, ok := prevTransform.(map[string]interface{}); ok {
			delete(prevObject, "scriptPath")
		}
	}
	if nextTransform, ok := nextData["transform"]; ok {
		if nextObject, ok := nextTransform.(map[string]interface{}); ok {
			delete(nextObject, "scriptPath")
		}
	}
	if prevReadme, ok := prevData["readme"]; ok {
		if prevObject, ok := prevReadme.(map[string]interface{}); ok {
			delete(prevObject, "scriptPath")
		}
	}
	if nextReadme, ok := nextData["readme"]; ok {
		if nextObject, ok := nextReadme.(map[string]interface{}); ok {
			delete(nextObject, "scriptPath")
		}
	}
	if prevMeta, ok := prevData["meta"]; ok {
		if prevObject, ok := prevMeta.(map[string]interface{}); ok {
			delete(prevObject, "path")
			delete(prevObject, "qri")
		}
	}
	if nextMeta, ok := nextData["meta"]; ok {
		if nextObject, ok := nextMeta.(map[string]interface{}); ok {
			delete(nextObject, "path")
			delete(nextObject, "qri")
		}
	}

	var prevChecksum, nextChecksum string

	if prevStructure, ok := prevData["structure"]; ok {
		if prevObject, ok := prevStructure.(map[string]interface{}); ok {
			if checksum, ok := prevObject["checksum"].(string); ok {
				prevChecksum = checksum
			}
			delete(prevObject, "checksum")
			delete(prevObject, "entries")
			delete(prevObject, "length")
			delete(prevObject, "depth")
			delete(prevObject, "path")
		}
	}
	if nextStructure, ok := nextData["structure"]; ok {
		if nextObject, ok := nextStructure.(map[string]interface{}); ok {
			if checksum, ok := nextObject["checksum"].(string); ok {
				nextChecksum = checksum
			}
			delete(nextObject, "checksum")
			delete(nextObject, "entries")
			delete(nextObject, "length")
			delete(nextObject, "depth")
			delete(nextObject, "path")
		}
	}

	// If the body is too big to diff, compare the checksums. If they differ, assume the
	// body has changed.
	assumeBodyChanged := false
	if bodyAct == BodyTooBig {
		prevBody = nil
		nextBody = nil
		log.Debugw("checking checksum equality", "prev", prevChecksum, "next", nextChecksum)
		if prevChecksum != nextChecksum {
			assumeBodyChanged = true
		}
	}

	var headDiff, bodyDiff deepdiff.Deltas
	var bodyStat *deepdiff.Stats

	// Diff the head and body separately. This allows accurate stats when figuring out how much
	// of the body has changed.
	headDiff, _, err = deepdiff.New().StatDiff(ctx, prevData, nextData)
	if err != nil {
		return "", "", err
	}
	if prevBody != nil && nextBody != nil {
		log.Debugf("calculating body statDiff type(prevBody)=%T type(nextBody)=%T", prevBody, nextBody)
		bodyDiff, bodyStat, err = deepdiff.New().StatDiff(ctx, prevBody, nextBody)
		if err != nil {
			log.Debugf("error calculating body statDiff: %q", err)
			return "", "", err
		}
	}

	shortTitle, longMessage := friendly.DiffDescriptions(headDiff, bodyDiff, bodyStat, assumeBodyChanged)
	if shortTitle == "" {
		if forceIfNoChanges {
			return "forced update", "forced update", nil
		}
		log.Debugw("generateCommitDescriptions", "err", ErrNoChanges)
		return "", "", ErrNoChanges
	}

	log.Debugw("generateCommitDescriptions", "shortTitle", shortTitle, "message", longMessage, "bodyChanged", assumeBodyChanged)
	return shortTitle, longMessage, nil
}
