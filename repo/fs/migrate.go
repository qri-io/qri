package fsrepo

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/qri-io/qri/repo"
)

// maybeCreateFlatbufferRefsFile creates a flatbuffer from an existing ds_refs
// json file. repo files used to be stored as json, and we're moving to
// flatbuffers
// TODO (b5) - we should consider keeping both JSON and flatbuffer records for a
// few releases if the json ds_ref.json file exists.
func maybeCreateFlatbufferRefsFile(repoPath string) (migrated bool, err error) {
	fbPath := filepath.Join(repoPath, Filepath(FileRefs))
	if _, err := os.Stat(fbPath); os.IsNotExist(err) {
		jsonPath := filepath.Join(repoPath, Filepath(FileJSONRefs))
		if jsonData, err := ioutil.ReadFile(jsonPath); err == nil {
			jsonRefs := repo.Refs{}
			if err = json.Unmarshal(jsonData, &jsonRefs); err == nil {
				return true, ioutil.WriteFile(fbPath, jsonRefs.FlatbufferBytes(), os.ModePerm)
			}
		}
	}
	return false, nil
}
