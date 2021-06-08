package base

import (
	"context"
	"io/ioutil"

	"github.com/microcosm-cc/bluemonday"
	"github.com/qri-io/qfs"
	"github.com/russross/blackfriday/v2"
)

// RenderReadme converts the markdown from the file into html.
func RenderReadme(ctx context.Context, file qfs.File) ([]byte, error) {
	data, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}
	r := blackfriday.NewHTMLRenderer(blackfriday.HTMLRendererParameters{
		Flags: blackfriday.CommonHTMLFlags | blackfriday.NoreferrerLinks | blackfriday.NoopenerLinks | blackfriday.HrefTargetBlank,
	})
	unsafe := blackfriday.Run(data, blackfriday.WithRenderer(r))
	htmlBytes := bluemonday.UGCPolicy().SanitizeBytes(unsafe)
	return htmlBytes, nil
}
