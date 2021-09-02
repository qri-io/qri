package lib

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/base/params"
	"github.com/qri-io/qri/dscache/build"
	"github.com/qri-io/qri/dsref"
	qhttp "github.com/qri-io/qri/lib/http"
	"github.com/qri-io/qri/profile"
	reporef "github.com/qri-io/qri/repo/ref"
)

// CollectionMethods lists a user's datasets. Datasets in a collection consist
// of datasets the user has created and other datasets the user has pulled.
//
// Collections are local. The same user's collection on one qri node will
// often be different from another node, depending on what datasets have been
// created, pushed, or pulled to that node
type CollectionMethods struct {
	d dispatcher
}

// Name returns the name of this method group
func (m CollectionMethods) Name() string {
	return "collection"
}

// Attributes defines attributes for each method
func (m CollectionMethods) Attributes() map[string]AttributeSet {
	return map[string]AttributeSet{
		"list":        {Endpoint: qhttp.AEList, HTTPVerb: "POST"},
		"listrawrefs": {Endpoint: qhttp.DenyHTTP},
		"get":         {Endpoint: qhttp.AECollectionGet, HTTPVerb: "POST"},
	}
}

// ErrListWarning is a warning that can occur while listing
var ErrListWarning = base.ErrUnlistableReferences

// List gets the reflist for either the local repo or a peer
func (m CollectionMethods) List(ctx context.Context, p *ListParams) ([]dsref.VersionInfo, error) {
	got, _, err := m.d.Dispatch(ctx, dispatchMethodName(m, "list"), p)
	if res, ok := got.([]dsref.VersionInfo); ok {
		return res, err
	}
	return nil, dispatchReturnError(got, err)
}

// ListRawRefs gets the list of raw references as string
func (m CollectionMethods) ListRawRefs(ctx context.Context, p *EmptyParams) (string, error) {
	got, _, err := m.d.Dispatch(ctx, dispatchMethodName(m, "listrawrefs"), p)
	if res, ok := got.(string); ok {
		return res, err
	}
	return "", dispatchReturnError(got, err)
}

// CollectionGetParams defines parameters for looking up the head of a dataset from the collection
type CollectionGetParams struct {
	Ref    string `json:"ref"`
	InitID string `json:"initID"`
}

// Validate returns an error if CollectionGetParams fields are in an invalid state
func (p *CollectionGetParams) Validate() error {
	if p.Ref == "" && p.InitID == "" {
		return fmt.Errorf("either ref or initID are required")
	}
	return nil
}

// Get gets the head of a dataset as a VersionInfo from the collection
func (m CollectionMethods) Get(ctx context.Context, p *CollectionGetParams) (*dsref.VersionInfo, error) {
	got, _, err := m.d.Dispatch(ctx, dispatchMethodName(m, "get"), p)
	if res, ok := got.(*dsref.VersionInfo); ok {
		return res, err
	}
	return nil, dispatchReturnError(got, err)
}

// collectionImpl holds the method implementations for CollectionMethods
type collectionImpl struct{}

// List gets the reflist for either the local repo or a peer
func (collectionImpl) List(scope scope, p *ListParams) ([]dsref.VersionInfo, error) {
	if s := scope.CollectionSet(); s != nil {
		lp := params.List{
			OrderBy: []string{p.OrderBy},
			Offset:  p.Offset,
			Limit:   p.Limit,
		}
		return s.List(scope.ctx, scope.ActiveProfile().ID, lp)
	}
	// TODO(dustmop): When List is converted to use scope, get the ProfileID from
	// the scope if the user is authorized to only view their own datasets, as opposed
	// to the full collection that exists in this node's repository.
	restrictPid := ""

	// ensure valid limit value
	if p.Limit <= 0 {
		p.Limit = 25
	}
	// ensure valid offset value
	if p.Offset < 0 {
		p.Offset = 0
	}

	reqProfile := scope.Repo().Profiles().Owner(scope.Context())
	listProfile, err := getProfile(scope.Context(), scope.Repo().Profiles(), reqProfile.ID.Encode(), p.Username)
	if err != nil {
		return nil, err
	}

	// If the list operation leads to a warning, store it in this var
	var listWarning error

	var infos []dsref.VersionInfo
	if scope.UseDscache() {
		c := scope.Dscache()
		if c.IsEmpty() {
			log.Infof("building dscache from repo's logbook, profile, and dsref")
			built, err := build.DscacheFromRepo(scope.Context(), scope.Repo())
			if err != nil {
				return nil, err
			}
			err = c.Assign(built)
			if err != nil {
				log.Error(err)
			}
		}
		refs, err := c.ListRefs()
		if err != nil {
			return nil, err
		}
		// Filter references so that only with a matching name are returned
		if p.Term != "" {
			matched := make([]reporef.DatasetRef, len(refs))
			count := 0
			for _, ref := range refs {
				if strings.Contains(ref.AliasString(), p.Term) {
					matched[count] = ref
					count++
				}
			}
			refs = matched[:count]
		}
		// Filter references by skipping to the correct offset
		if p.Offset > len(refs) {
			refs = []reporef.DatasetRef{}
		} else {
			refs = refs[p.Offset:]
		}
		// Filter references by limiting how many are returned
		if p.Limit < len(refs) {
			refs = refs[:p.Limit]
		}
		// Convert old style DatasetRef list to VersionInfo list.
		// TODO(dustmop): Remove this and convert lower-level functions to return []VersionInfo.
		infos = make([]dsref.VersionInfo, len(refs))
		for i, r := range refs {
			infos[i] = reporef.ConvertToVersionInfo(&r)
		}
	} else if listProfile.Peername == "" || reqProfile.Peername == listProfile.Peername {
		infos, err = base.ListDatasets(scope.Context(), scope.Repo(), p.Term, restrictPid, p.Offset, p.Limit, p.Public, true)
		if errors.Is(err, ErrListWarning) {
			// This warning can happen when there's conflicts between usernames and
			// profileIDs. This type of conflict should not break listing functionality.
			// Instead, store the warning and treat it as non-fatal.
			listWarning = err
			err = nil
		}
	} else {
		return nil, fmt.Errorf("listing datasets on a peer is not implemented")
	}
	if err != nil {
		return nil, err
	}

	if listWarning != nil {
		// If there was a warning listing the datasets, we should still return the list
		// itself. The caller should handle this warning by simply printing it, but this
		// shouldn't break the `list` functionality.
		return infos, listWarning
	}

	return infos, nil
}

func getProfile(ctx context.Context, pros profile.Store, idStr, peername string) (pro *profile.Profile, err error) {
	if idStr == "" {
		// TODO(b5): we're handling the "me" keyword here, should be handled as part of
		// request scope construction
		if peername == "me" {
			return pros.Owner(ctx), nil
		}
		return profile.ResolveUsername(ctx, pros, peername)
	}

	id, err := profile.IDB58Decode(idStr)
	if err != nil {
		log.Debugw("decoding profile ID", "err", err)
		return nil, err
	}
	return pros.GetProfile(ctx, id)
}

// ListRawRefs gets the list of raw references as string
func (collectionImpl) ListRawRefs(scope scope, p *EmptyParams) (string, error) {
	text := ""
	if scope.UseDscache() {
		c := scope.Dscache()
		if c == nil || c.IsEmpty() {
			return "", fmt.Errorf("repo: dscache not found")
		}
		text = c.VerboseString(true)
		return text, nil
	}
	return base.RawDatasetRefs(scope.Context(), scope.ActiveProfile().ID, scope.CollectionSet())
}

// Get gets the head of a dataset as a VersionInfo from the collection
func (collectionImpl) Get(scope scope, p *CollectionGetParams) (*dsref.VersionInfo, error) {
	s := scope.CollectionSet()
	if s == nil {
		return nil, fmt.Errorf("no collection")
	}
	if p.InitID == "" {
		ref, _, err := scope.ParseAndResolveRef(scope.Context(), p.Ref)
		if err != nil {
			return nil, err
		}
		p.InitID = ref.InitID
	}
	return s.Get(scope.Context(), scope.ActiveProfile().ID, p.InitID)
}
