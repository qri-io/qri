package fsi

import (
	"io/ioutil"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// FilesysObservation represents the result of observing what exists in a FSI directory.
type FilesysObservation struct {
	Held map[string]*ComponentHolder
}

// ComponentHolder contains all the state concerning a simple component. Operates in a lazy manner.
type ComponentHolder struct {
	Name           string
	Value          interface{}
	ModTime        time.Time
	ProblemKind    string
	ProblemMessage string
	SourceFile     string
	IsLoaded       bool
	OwnFile        bool
	Format         string
}

// TODO(dlong): Perhaps add a Holder interface that is implemented by MetaHolder, BodyHolder etc
// concrete types, with each of them using Name() to return their name. This would also come
// with a IsEqual() method and Write() method, etc.

// ObserveDirectory reads the files at the given directory and returns an observation. The
// resulting object has stat'ed each file, and has their mtimes, but no files have been read
// from disk. Conflicting files (such as both a "body.csv" and "body.json") will cause the
// "ProblemKind" and "ProblemMessage" fields to be set. Other conflicts may also exist, such
// as "meta" being in both "dataset.json" and "meta.json", but this function does not detect
// these kinds of problems because it does not read any files.
func ObserveDirectory(path string) (FilesysObservation, error) {
	knownFilenames := GetKnownFilenames()
	fo := FilesysObservation{Held: make(map[string]*ComponentHolder)}
	finfos, err := ioutil.ReadDir(path)
	if err != nil {
		return fo, err
	}
	for _, fi := range finfos {
		ext := filepath.Ext(fi.Name())
		componentName := strings.TrimSuffix(fi.Name(), ext)
		allowedExtensions, ok := knownFilenames[componentName]
		if ok {
			if sliceContains(allowedExtensions, ext) {
				absPath, _ := filepath.Abs(filepath.Join(path, fi.Name()))
				// Check for conflict
				if elem, ok := fo.Held[componentName]; ok {
					elem.ProblemKind = "conflict"
					// Collect a message containing the paths of conflicting files
					msg := elem.ProblemMessage
					if msg == "" {
						msg = filepath.Base(elem.SourceFile)
					}
					// Sort the problem files so that the message is deterministic
					conflictFiles := append(strings.Split(msg, " "), filepath.Base(absPath))
					sort.Strings(conflictFiles)
					elem.ProblemMessage = strings.Join(conflictFiles, " ")
					continue
				}
				fo.Held[componentName] = &ComponentHolder{
					Name:       componentName,
					ModTime:    fi.ModTime(),
					SourceFile: absPath,
					OwnFile:    true,
					Format:     removeDotPrefix(ext),
				}
			}
		}
	}
	return fo, nil
}

// ExpandObservation will read whatever is necessary in order to discover all of the components
// that exist within this observation. For example, is a "dataset" exists, it will be read to find
// out if it contains a "meta", a "structure", etc.
func ExpandObservation(fo *FilesysObservation) {
	// TODO: If "dataset" exists, open it and extract components. If "structure" exists, open it
	// and extract "schema".
}

// GetKnownFilenames returns a map containing all possible filenames (filebase and extension) for
// any file that can represent a component of a dataset.
func GetKnownFilenames() map[string][]string {
	componentExtensionTypes := []string{".json", ".yml", ".yaml"}
	bodyExtensionTypes := []string{".csv", ".json", ".cbor", ".xlsx"}
	return map[string][]string{
		"dataset":   componentExtensionTypes,
		"commit":    componentExtensionTypes,
		"meta":      componentExtensionTypes,
		"structure": componentExtensionTypes,
		"schema":    componentExtensionTypes,
		"viz":       []string{".html"},
		"transform": []string{".star"},
		"body":      bodyExtensionTypes,
	}
}

func removeDotPrefix(text string) string {
	if strings.HasPrefix(text, ".") {
		return text[1:]
	}
	return text
}

func sliceContains(subject []string, needle string) bool {
	for _, elem := range subject {
		if elem == needle {
			return true
		}
	}
	return false
}
