package dscache

import (
	"context"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	flatbuffers "github.com/google/flatbuffers/go"
	golog "github.com/ipfs/go-log"
	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/dscache/dscachefb"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/repo/profile"
	reporef "github.com/qri-io/qri/repo/ref"
)

var (
	log = golog.Logger("dscache")
	// lengthOfProfileID is the expected length of a valid profileID
	lengthOfProfileID = 46
	// ErrNoDscache is returned when methods are called on a non-existant Dscache
	ErrNoDscache = fmt.Errorf("dscache: does not exist")
	// ErrInvalidProfileID is returned when an invalid profileID is given to dscache
	ErrInvalidProfileID = fmt.Errorf("invalid profileID")
)

// Dscache represents an in-memory serialized dscache flatbuffer
type Dscache struct {
	Filename            string
	Root                *dscachefb.Dscache
	Buffer              []byte
	CreateNewEnabled    bool
	ProfileIDToUsername map[string]string
	DefaultUsername     string
}

// NewDscache will construct a dscache from the given filename, or will construct an empty dscache
// that will save to the given filename. Using an empty filename will disable loading and saving
func NewDscache(ctx context.Context, fsys qfs.Filesystem, bus event.Bus, username, filename string) *Dscache {
	cache := Dscache{Filename: filename}
	f, err := fsys.Get(ctx, filename)
	if err == nil {
		// Ignore error, as dscache loading is optional
		defer f.Close()
		buffer, err := ioutil.ReadAll(f)
		if err != nil {
			log.Error(err)
		} else {
			root := dscachefb.GetRootAsDscache(buffer, 0)
			cache = Dscache{Filename: filename, Root: root, Buffer: buffer}
		}
	}
	cache.DefaultUsername = username
	bus.Subscribe(cache.handler,
		event.ETDatasetNameInit,
		event.ETDatasetCommitChange,
		event.ETDatasetDeleteAll,
		event.ETDatasetRename,
		event.ETDatasetCreateLink)

	return &cache
}

// IsEmpty returns whether the dscache has any constructed data in it
func (d *Dscache) IsEmpty() bool {
	if d == nil {
		return true
	}
	return d.Root == nil
}

// Assign assigns the data from one dscache to this one
func (d *Dscache) Assign(other *Dscache) error {
	if d == nil {
		return ErrNoDscache
	}
	d.Root = other.Root
	d.Buffer = other.Buffer
	return d.save()
}

// VerboseString is a convenience function that returns a readable string, for testing and debugging
func (d *Dscache) VerboseString(showEmpty bool) string {
	if d.IsEmpty() {
		return "dscache: cannot not stringify an empty dscache"
	}
	out := strings.Builder{}
	out.WriteString("Dscache:\n")
	out.WriteString(" Dscache.Users:\n")
	for i := 0; i < d.Root.UsersLength(); i++ {
		userAssoc := dscachefb.UserAssoc{}
		d.Root.Users(&userAssoc, i)
		username := userAssoc.Username()
		profileID := userAssoc.ProfileID()
		fmt.Fprintf(&out, " %2d) user=%s profileID=%s\n", i, username, profileID)
	}
	out.WriteString(" Dscache.Refs:\n")
	for i := 0; i < d.Root.RefsLength(); i++ {
		r := dscachefb.RefEntryInfo{}
		d.Root.Refs(&r, i)
		fmt.Fprintf(&out, ` %2d) initID        = %s
     profileID     = %s
     topIndex      = %d
     cursorIndex   = %d
     prettyName    = %s
`, i, r.InitID(), r.ProfileID(), r.TopIndex(), r.CursorIndex(), r.PrettyName())
		indent := "     "
		if len(r.MetaTitle()) != 0 || showEmpty {
			fmt.Fprintf(&out, "%smetaTitle     = %s\n", indent, r.MetaTitle())
		}
		if len(r.ThemeList()) != 0 || showEmpty {
			fmt.Fprintf(&out, "%sthemeList     = %s\n", indent, r.ThemeList())
		}
		if r.BodySize() != 0 || showEmpty {
			fmt.Fprintf(&out, "%sbodySize      = %d\n", indent, r.BodySize())
		}
		if r.BodyRows() != 0 || showEmpty {
			fmt.Fprintf(&out, "%sbodyRows      = %d\n", indent, r.BodyRows())
		}
		if r.CommitTime() != 0 || showEmpty {
			fmt.Fprintf(&out, "%scommitTime    = %d\n", indent, r.CommitTime())
		}
		if r.NumErrors() != 0 || showEmpty {
			fmt.Fprintf(&out, "%snumErrors     = %d\n", indent, r.NumErrors())
		}
		if len(r.HeadRef()) != 0 || showEmpty {
			fmt.Fprintf(&out, "%sheadRef       = %s\n", indent, r.HeadRef())
		}
		if len(r.FsiPath()) != 0 || showEmpty {
			fmt.Fprintf(&out, "%sfsiPath       = %s\n", indent, r.FsiPath())
		}
	}
	return out.String()
}

// ListRefs returns references to each dataset in the cache
func (d *Dscache) ListRefs() ([]reporef.DatasetRef, error) {
	if d.IsEmpty() {
		return nil, ErrNoDscache
	}
	d.ensureProToUserMap()
	refs := make([]reporef.DatasetRef, 0, d.Root.RefsLength())
	for i := 0; i < d.Root.RefsLength(); i++ {
		refCache := dscachefb.RefEntryInfo{}
		d.Root.Refs(&refCache, i)

		proIDStr := string(refCache.ProfileID())
		profileID, err := profile.NewB58ID(proIDStr)
		if err != nil {
			log.Errorf("could not parse profileID %q", proIDStr)
		}
		username, ok := d.ProfileIDToUsername[proIDStr]
		if !ok {
			log.Errorf("no username associated with profileID %q", proIDStr)
		}

		refs = append(refs, reporef.DatasetRef{
			Peername:  username,
			ProfileID: profileID,
			Name:      string(refCache.PrettyName()),
			Path:      string(refCache.HeadRef()),
			FSIPath:   string(refCache.FsiPath()),
			Dataset: &dataset.Dataset{
				Meta: &dataset.Meta{
					Title: string(refCache.MetaTitle()),
				},
				Structure: &dataset.Structure{
					ErrCount: int(refCache.NumErrors()),
					Entries:  int(refCache.BodyRows()),
					Length:   int(refCache.BodySize()),
				},
				Commit:      &dataset.Commit{},
				NumVersions: int(refCache.TopIndex()),
			},
		})
	}
	return refs, nil
}

// ResolveRef finds the identifier for a dataset reference
// implements dsref.Resolver interface
func (d *Dscache) ResolveRef(ctx context.Context, ref *dsref.Ref) (string, error) {
	// NOTE: isEmpty is nil-callable. important b/c ResolveRef must be nil-callable
	if d.IsEmpty() {
		return "", dsref.ErrRefNotFound
	}

	vi, err := d.LookupByName(*ref)
	if err != nil {
		return "", dsref.ErrRefNotFound
	}

	ref.InitID = vi.InitID
	ref.ProfileID = vi.ProfileID
	if ref.Path == "" {
		ref.Path = vi.Path
	}

	return "", nil
}

// LookupByName looks up a dataset by dsref and returns the latest VersionInfo if found
func (d *Dscache) LookupByName(ref dsref.Ref) (*dsref.VersionInfo, error) {
	// Convert the username into a profileID
	for i := 0; i < d.Root.UsersLength(); i++ {
		userAssoc := dscachefb.UserAssoc{}
		d.Root.Users(&userAssoc, i)
		username := userAssoc.Username()
		profileID := userAssoc.ProfileID()
		if ref.Username == string(username) {
			// TODO(dustmop): Switch off of profileID to a stable ID (that handle key rotations)
			// based upon the Logbook creation of a user's profile.
			ref.ProfileID = string(profileID)
			break
		}
	}
	if ref.ProfileID == "" {
		return nil, fmt.Errorf("unknown username %q", ref.Username)
	}
	// Lookup the info, given the profileID/dsname
	for i := 0; i < d.Root.RefsLength(); i++ {
		r := dscachefb.RefEntryInfo{}
		d.Root.Refs(&r, i)
		if string(r.ProfileID()) == ref.ProfileID && string(r.PrettyName()) == ref.Name {
			info := convertEntryToVersionInfo(&r)
			return &info, nil
		}
	}
	return nil, fmt.Errorf("dataset ref not found %s/%s", ref.Username, ref.Name)
}

func (d *Dscache) validateProfileID(profileID string) bool {
	return len(profileID) == lengthOfProfileID
}

func (d *Dscache) handler(_ context.Context, t event.Type, payload interface{}) error {
	act, ok := payload.(event.DsChange)
	if !ok {
		log.Error("dscache got an event with a payload that isn't a event.DsChange type: %v", payload)
		return nil
	}

	switch t {
	case event.ETDatasetNameInit:
		if err := d.updateInitDataset(act); err != nil && err != ErrNoDscache {
			log.Error(err)
		}
	case event.ETDatasetCommitChange:
		if err := d.updateChangeCursor(act); err != nil && err != ErrNoDscache {
			log.Error(err)
		}
	case event.ETDatasetDeleteAll:
		if err := d.updateDeleteDataset(act); err != nil && err != ErrNoDscache {
			log.Error(err)
		}
	case event.ETDatasetRename:
		// TODO(dustmop): Handle renames
	case event.ETDatasetCreateLink:
		if err := d.updateCreateLink(act); err != nil && err != ErrNoDscache {
			log.Error(err)
		}
	}

	return nil
}

func (d *Dscache) updateInitDataset(act event.DsChange) error {
	if d.IsEmpty() {
		// Only create a new dscache if that feature is enabled. This way no one is forced to
		// use dscache without opting in.
		if !d.CreateNewEnabled {
			return nil
		}

		if !d.validateProfileID(act.ProfileID) {
			return ErrInvalidProfileID
		}

		builder := NewBuilder()
		builder.AddUser(act.Username, act.ProfileID)
		builder.AddDsVersionInfo(dsref.VersionInfo{
			InitID:    act.InitID,
			ProfileID: act.ProfileID,
			Name:      act.PrettyName,
		})
		cache := builder.Build()
		d.Assign(cache)
		return nil
	}
	builder := NewBuilder()
	// copy users
	for i := 0; i < d.Root.UsersLength(); i++ {
		up := dscachefb.UserAssoc{}
		d.Root.Users(&up, i)
		builder.AddUser(string(up.Username()), string(up.ProfileID()))
	}
	// copy ds versions
	for i := 0; i < d.Root.UsersLength(); i++ {
		r := dscachefb.RefEntryInfo{}
		d.Root.Refs(&r, i)
		builder.AddDsVersionInfoWithIndexes(convertEntryToVersionInfo(&r), int(r.TopIndex()), int(r.CursorIndex()))
	}
	// Add new ds version info
	builder.AddDsVersionInfo(dsref.VersionInfo{
		InitID:    act.InitID,
		ProfileID: act.ProfileID,
		Name:      act.PrettyName,
	})
	cache := builder.Build()
	d.Assign(cache)
	return nil
}

// Copy the entire dscache, except for the matching entry, rebuild that one to modify it
func (d *Dscache) updateChangeCursor(act event.DsChange) error {
	if d.IsEmpty() {
		return ErrNoDscache
	}
	// Flatbuffers for go do not allow mutation (for complex types like strings). So we construct
	// a new flatbuffer entirely, copying the old one while replacing the entry we care to change.
	builder := flatbuffers.NewBuilder(0)
	users := d.copyUserAssociationList(builder)
	refs := d.copyReferenceListWithReplacement(
		builder,
		// Function to match the entry we're looking to replace
		func(r *dscachefb.RefEntryInfo) bool {
			return string(r.InitID()) == act.InitID
		},
		// Function to replace the matching entry
		func(refStartMutationFunc func(builder *flatbuffers.Builder)) {
			var metaTitle flatbuffers.UOffsetT
			if act.Info != nil {
				metaTitle = builder.CreateString(act.Info.MetaTitle)
			}
			hashRef := builder.CreateString(string(act.HeadRef))
			// Start building a ref object, by mutating an existing ref object.
			refStartMutationFunc(builder)
			// Add only the fields we want to change.
			dscachefb.RefEntryInfoAddTopIndex(builder, int32(act.TopIndex))
			dscachefb.RefEntryInfoAddCursorIndex(builder, int32(act.TopIndex))
			if act.Info != nil {
				dscachefb.RefEntryInfoAddMetaTitle(builder, metaTitle)
				dscachefb.RefEntryInfoAddCommitTime(builder, act.Info.CommitTime.Unix())
				dscachefb.RefEntryInfoAddBodySize(builder, int64(act.Info.BodySize))
				dscachefb.RefEntryInfoAddBodyRows(builder, int32(act.Info.BodyRows))
				dscachefb.RefEntryInfoAddNumErrors(builder, int32(act.Info.NumErrors))
			}
			dscachefb.RefEntryInfoAddHeadRef(builder, hashRef)
			// Don't call RefEntryInfoEnd, that is handled by copyReferenceListWithReplacement
		},
	)
	root, serialized := d.finishBuilding(builder, users, refs)
	d.Root = root
	d.Buffer = serialized
	return d.save()
}

// Copy the entire dscache, except leave out the matching entry.
func (d *Dscache) updateDeleteDataset(act event.DsChange) error {
	if d.IsEmpty() {
		return ErrNoDscache
	}
	// Flatbuffers for go do not allow mutation (for complex types like strings). So we construct
	// a new flatbuffer entirely, copying the old one while omitting the entry we want to remove.
	builder := flatbuffers.NewBuilder(0)
	users := d.copyUserAssociationList(builder)
	refs := d.copyReferenceListWithReplacement(
		builder,
		func(r *dscachefb.RefEntryInfo) bool {
			return string(r.InitID()) == act.InitID
		},
		// Pass a nil function, so the matching entry is not replaced, it is omitted
		nil,
	)
	root, serialized := d.finishBuilding(builder, users, refs)
	d.Root = root
	d.Buffer = serialized
	return d.save()
}

// Copy the entire dscache, except for the matching entry, which is copied then assigned an fsiPath
func (d *Dscache) updateCreateLink(act event.DsChange) error {
	if d.IsEmpty() {
		return ErrNoDscache
	}
	// Flatbuffers for go do not allow mutation (for complex types like strings). So we construct
	// a new flatbuffer entirely, copying the old one while replacing the entry we care to change.
	builder := flatbuffers.NewBuilder(0)
	users := d.copyUserAssociationList(builder)
	refs := d.copyReferenceListWithReplacement(
		builder,
		// Function to match the entry we're looking to replace
		func(r *dscachefb.RefEntryInfo) bool {
			if act.InitID != "" {
				return string(r.InitID()) == act.InitID
			}
			return d.DefaultUsername == act.Username && string(r.PrettyName()) == act.PrettyName
		},
		// Function to replace the matching entry
		func(refStartMutationFunc func(builder *flatbuffers.Builder)) {
			fsiDir := builder.CreateString(string(act.Dir))
			// Start building a ref object, by mutating an existing ref object.
			refStartMutationFunc(builder)
			// For this kind of update, only the fsiDir is modified
			dscachefb.RefEntryInfoAddFsiPath(builder, fsiDir)
			// Don't call RefEntryInfoEnd, that is handled by copyReferenceListWithReplacement
		},
	)
	root, serialized := d.finishBuilding(builder, users, refs)
	d.Root = root
	d.Buffer = serialized
	return d.save()
}

func convertEntryToVersionInfo(r *dscachefb.RefEntryInfo) dsref.VersionInfo {
	return dsref.VersionInfo{
		InitID:      string(r.InitID()),
		ProfileID:   string(r.ProfileID()),
		Name:        string(r.PrettyName()),
		Path:        string(r.HeadRef()),
		Published:   r.Published(),
		Foreign:     r.Foreign(),
		MetaTitle:   string(r.MetaTitle()),
		ThemeList:   string(r.ThemeList()),
		BodySize:    int(r.BodySize()),
		BodyRows:    int(r.BodyRows()),
		BodyFormat:  string(r.BodyFormat()),
		NumErrors:   int(r.NumErrors()),
		CommitTime:  time.Unix(r.CommitTime(), 0),
		NumVersions: int(r.NumVersions()),
		FSIPath:     string(r.FsiPath()),
	}
}

func (d *Dscache) ensureProToUserMap() {
	if d.ProfileIDToUsername != nil {
		return
	}
	d.ProfileIDToUsername = make(map[string]string)
	for i := 0; i < d.Root.UsersLength(); i++ {
		userAssoc := dscachefb.UserAssoc{}
		d.Root.Users(&userAssoc, i)
		username := userAssoc.Username()
		profileID := userAssoc.ProfileID()
		d.ProfileIDToUsername[string(profileID)] = string(username)
	}
}

// save writes the serialized bytes to the given filename
func (d *Dscache) save() error {
	if d.Filename == "" {
		log.Infof("dscache: no filename set, will not save")
		return nil
	}
	return ioutil.WriteFile(d.Filename, d.Buffer, 0644)
}
