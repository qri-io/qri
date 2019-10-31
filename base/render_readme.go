package base

import (
	"context"
	"io/ioutil"

	"github.com/microcosm-cc/bluemonday"
	"github.com/qri-io/qfs"
	"github.com/russross/blackfriday/v2"
)

// RenderReadme converts the markdown from the file into html.
func RenderReadme(ctx context.Context, file qfs.File) (string, error) {
	data, err := ioutil.ReadAll(file)
	if err != nil {
		return "", err
	}
	unsafe := blackfriday.Run(data)
	htmlBytes := bluemonday.UGCPolicy().SanitizeBytes(unsafe)
	return string(htmlBytes), nil
}
