package remote

import (
	"context"
	"fmt"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/repo"
)

// Feeds accesses streams of dataset VersionInfo's to browse. Feeds should be
// named by their defining characteristic (eg: "popular", "recent", etc.) to
// distinguish their intention. Feed names must be unique.
//
// The precise behaviour of feeds if left up to the responder. Remotes can make
// any number of feeds available, and update those feeds with any frequency.
// A remote may construct feeds of datasets that they don't have data for,
// simply to assist in dataset discovery.
//
// The userID argument is planned for future use. The Qri roadmap includes plans
// to implement access control some day, providing an identifier for the user
// requesting a feed will allow the provider to tailor feeds to show datasets
// that user may have priviledged access to.
type Feeds interface {
	// Feeds returns a set of feeds keyed by name, the number of results in each
	// feed, and the number of feeds themselves is up to the server
	Feeds(ctx context.Context, userID string) (map[string][]dsref.VersionInfo, error)
	// Feed fetches a bounded set of VersionInfos for a given feed name
	Feed(ctx context.Context, userID, name string, offset, limit int) ([]dsref.VersionInfo, error)
}

// RepoFeeds implements the feed interface with a Repo
type RepoFeeds struct {
	repo.Repo
}

// assert at compile time that RepoFeeds implements the Feeds interface
var _ Feeds = (*RepoFeeds)(nil)

// Feeds returns a set of feeds keyed by name, fetching a few references for
// each available feed
func (rf RepoFeeds) Feeds(ctx context.Context, userID string) (map[string][]dsref.VersionInfo, error) {
	recent, err := rf.Feed(ctx, userID, "recent", 0, 10)
	if err != nil {
		return nil, err
	}
	return map[string][]dsref.VersionInfo{
		"recent": recent,
	}, nil
}

// Feed fetches a portion of an individual named feed
func (rf RepoFeeds) Feed(ctx context.Context, userID, name string, offset, limit int) ([]dsref.VersionInfo, error) {
	if name != "recent" {
		return nil, fmt.Errorf("unknown feed name '%s'", name)
	}

	refs, err := base.ListDatasets(ctx, rf.Repo, "", limit, offset, false, true, false)
	if err != nil {
		return nil, err
	}
	res := make([]dsref.VersionInfo, len(refs))
	for i, ref := range refs {
		ref.Dataset.Name = ref.Name
		ref.Dataset.Peername = ref.Peername
		res[i] = dsref.ConvertDatasetToVersionInfo(ref.Dataset)
	}

	return res, nil
}

// Previews is an interface for generating constant-size summaries of dataset
// data
type Previews interface {
	Preview(ctx context.Context, userID, refStr string) (*dataset.Dataset, error)
	PreviewComponent(ctx context.Context, userID, refStr, component string) (interface{}, error)
}

// LocalPreviews implements the previews interface with a local repo
type LocalPreviews struct {
	fs            qfs.Filesystem
	localResolver dsref.Resolver
}

// assert at compile time that LocalPreviews implements the Previews interface
var _ Previews = (*LocalPreviews)(nil)

// Preview gets a preview for a reference
func (rp LocalPreviews) Preview(ctx context.Context, _, refStr string) (*dataset.Dataset, error) {
	ref, err := dsref.Parse(refStr)
	if err != nil {
		return nil, err
	}

	if _, err := rp.localResolver.ResolveRef(ctx, &ref); err != nil {
		return nil, err
	}

	return base.CreatePreview(ctx, rp.fs, ref)
}

// PreviewComponent gets a component for a reference & component name
func (rp LocalPreviews) PreviewComponent(ctx context.Context, _, refStr, component string) (interface{}, error) {
	return nil, fmt.Errorf("not finished")
}
