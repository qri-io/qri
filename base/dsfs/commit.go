package dsfs

import (
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"time"

	crypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsio"
	"github.com/qri-io/deepdiff"
	"github.com/qri-io/qfs/cafs"
	"github.com/qri-io/qfs/localfs"
	"github.com/qri-io/qri/base/friendly"
	"github.com/qri-io/qri/base/toqtype"
)

// Timestamp is an function for getting commit timestamps
// timestamps MUST be stored in UTC time zone
var Timestamp = func() time.Time {
	return time.Now().UTC()
}

// BodyAction represents the action that should be taken to understand how the body changed
type BodyAction string

const (
	// BodyDefault is the default action: compare them to get how much changed
	BodyDefault = BodyAction("default")
	// BodySame means that the bodies are the same, no need to compare
	BodySame = BodyAction("same")
	// BodyTooBig means the body is too big to compare, and should be assumed to have changed
	BodyTooBig = BodyAction("too_big")
)

// loadCommit assumes the provided path is valid
func loadCommit(ctx context.Context, store cafs.Filestore, path string) (st *dataset.Commit, err error) {
	data, err := fileBytes(store.Get(ctx, path))
	if err != nil {
		log.Debug(err.Error())
		return nil, fmt.Errorf("error loading commit file: %s", err.Error())
	}
	return dataset.UnmarshalCommit(data)
}

// generateCommitTileAndMessage creates the commit and title, message
func generateCommitTitleAndMessage(ctx context.Context, store cafs.Filestore, privKey crypto.PrivKey, ds, prev *dataset.Dataset, bodyAct BodyAction, fileHint string, forceIfNoChanges bool) error {
	log.Debugf("generateCommitTitleAndMessage bodyAct=%s", bodyAct)
	shortTitle, longMessage, err := generateCommitDescriptions(ctx, store, ds, prev, bodyAct, forceIfNoChanges)
	if err != nil {
		log.Debugf("generateCommitDescriptions err: %s", err)
		return fmt.Errorf("error saving: %s", err)
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
func generateCommitDescriptions(ctx context.Context, store cafs.Filestore, ds, prev *dataset.Dataset, bodyAct BodyAction, forceIfNoChanges bool) (short, long string, err error) {
	if prev == nil || prev.IsEmpty() {
		return defaultCreatedDescription, defaultCreatedDescription, nil
	}
	log.Debug("generateCommitDescriptions")

	// Inline body if it is a reasonable size, to get message about how the body has changed.
	if bodyAct != BodySame {
		// If previous version had bodyfile, read it and assign it
		if prev.Structure != nil && prev.Structure.Length < BodySizeSmallEnoughToDiff {
			if prev.BodyFile() != nil {
				log.Debugf("inlining body file to calulate a diff")
				prevReader, err := dsio.NewEntryReader(prev.Structure, prev.BodyFile())
				if err == nil {
					prevBodyData, err := dsio.ReadAll(prevReader)
					if err == nil {
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
		err := prev.Transform.OpenScriptFile(ctx, store)
		if err != nil {
			log.Error("prev.Transform.ScriptPath %q open err: %s", prev.Transform.ScriptPath, err)
		} else {
			tfFile := prev.Transform.ScriptFile()
			prev.Transform.ScriptBytes, err = ioutil.ReadAll(tfFile)
			if err != nil {
				log.Error("prev.Transform.ScriptPath %q read err: %s", prev.Transform.ScriptPath, err)
			}
		}
	}
	if ds.Transform != nil && ds.Transform.ScriptPath != "" {
		log.Debugf("inlining next transform ScriptPath=%q", ds.Transform.ScriptPath)
		// TODO(dustmop): The ipfs filestore won't recognize local filepaths, we need to use
		// local here. Is there some way to have a cafs store that works with both?
		fs, err := localfs.NewFS(nil)
		if err != nil {
			log.Errorf("error setting up local fs: %s", err)
		}
		err = ds.Transform.OpenScriptFile(ctx, fs)
		if err != nil {
			log.Errorf("ds.Transform.ScriptPath %q open err: %s", ds.Transform.ScriptPath, err)
		} else {
			tfFile := ds.Transform.ScriptFile()
			ds.Transform.ScriptBytes, err = ioutil.ReadAll(tfFile)
			if err != nil {
				log.Errorf("ds.Transform.ScriptPath %q read err: %s", ds.Transform.ScriptPath, err)
			}
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
		err := prev.Readme.OpenScriptFile(ctx, store)
		if err != nil {
			log.Error("prev.Readme.ScriptPath %q open err: %s", prev.Readme.ScriptPath, err)
		} else {
			tfFile := prev.Readme.ScriptFile()
			prev.Readme.ScriptBytes, err = ioutil.ReadAll(tfFile)
			if err != nil {
				log.Error("prev.Readme.ScriptPath %q read err: %s", prev.Readme.ScriptPath, err)
			}
		}
	}
	if ds.Readme != nil && ds.Readme.ScriptPath != "" {
		log.Debugf("inlining next readme ScriptPath=%q", ds.Readme.ScriptPath)
		// TODO(dustmop): The ipfs filestore won't recognize local filepaths, we need to use
		// local here. Is there some way to have a cafs store that works with both?
		fs, err := localfs.NewFS(nil)
		if err != nil {
			log.Error("localfs.NewFS err: %s", err)
		}
		err = ds.Readme.OpenScriptFile(ctx, fs)
		if err != nil {
			log.Debugf("ds.Readme.ScriptPath %q open err: %s", ds.Readme.ScriptPath, err)
			err = nil
		} else {
			tfFile := ds.Readme.ScriptFile()
			ds.Readme.ScriptBytes, err = ioutil.ReadAll(tfFile)
			if err != nil {
				log.Errorf("ds.Readme.ScriptPath %q read err: %s", ds.Readme.ScriptPath, err)
			}
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

	if prevStructure, ok := prevData["structure"]; ok {
		if prevObject, ok := prevStructure.(map[string]interface{}); ok {
			delete(prevObject, "checksum")
			delete(prevObject, "entries")
			delete(prevObject, "length")
			delete(prevObject, "depth")
		}
	}
	if nextStructure, ok := nextData["structure"]; ok {
		if nextObject, ok := nextStructure.(map[string]interface{}); ok {
			delete(nextObject, "checksum")
			delete(nextObject, "entries")
			delete(nextObject, "length")
			delete(nextObject, "depth")
		}
	}

	// If the body is too big to diff, compare the checksums. If they differ, assume the
	// body has changed.
	assumeBodyChanged := false
	if bodyAct == BodyTooBig {
		prevBody = nil
		nextBody = nil
		assumeBodyChanged = true
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

	log.Debug("setting diff descriptions")
	shortTitle, longMessage := friendly.DiffDescriptions(headDiff, bodyDiff, bodyStat, assumeBodyChanged)
	if shortTitle == "" {
		if forceIfNoChanges {
			return "forced update", "forced update", nil
		}
		return "", "", fmt.Errorf("no changes")
	}

	log.Debugf("set friendly diff descriptions. shortTitle=%q", shortTitle)
	return shortTitle, longMessage, nil
}
