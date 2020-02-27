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
	dscachefb "github.com/qri-io/qri/dscache/dscachefb"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/logbook"
	"github.com/qri-io/qri/repo/profile"
	reporef "github.com/qri-io/qri/repo/ref"
)

var (
	log = golog.Logger("dscache")
	// ErrNoDscache is returned when methods are called on a non-existant Dscache
	ErrNoDscache = fmt.Errorf("dscache: does not exist")
)

// Dscache represents an in-memory serialized dscache flatbuffer
type Dscache struct {
	Filename            string
	Root                *dscachefb.Dscache
	Buffer              []byte
	CreateNewEnabled    bool
	ProfileIDToUsername map[string]string
}

// NewDscache will construct a dscache from the given filename, or will construct an empty dscache
// that will save to the given filename. Using an empty filename will disable loading and saving
func NewDscache(ctx context.Context, fsys qfs.Filesystem, book *logbook.Book, filename string) *Dscache {
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
	if book != nil {
		book.Observe(cache.update)
	}
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
		if len(r.CommitTitle()) != 0 || showEmpty {
			fmt.Fprintf(&out, "%scommitTitle   = %s\n", indent, r.CommitTitle())
		}
		if len(r.CommitMessage()) != 0 || showEmpty {
			fmt.Fprintf(&out, "%scommitMessage = %s\n", indent, r.CommitMessage())
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

func (d *Dscache) update(act *logbook.Action) {
	switch act.Type {
	case logbook.ActionDatasetNameInit:
		if err := d.updateInitDataset(act); err != nil && err != ErrNoDscache {
			log.Error(err)
		}
	case logbook.ActionDatasetChange:
		if err := d.updateMoveCursor(act); err != nil && err != ErrNoDscache {
			log.Error(err)
		}
	}
}

func (d *Dscache) updateInitDataset(act *logbook.Action) error {
	if d.IsEmpty() {
		// Only create a new dscache if that feature is enabled. This way no one is forced to
		// use dscache without opting in.
		if !d.CreateNewEnabled {
			return nil
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

// Update modifies the dscache according to the provided action.
func (d *Dscache) updateMoveCursor(act *logbook.Action) error {
	if d.IsEmpty() {
		return ErrNoDscache
	}
	// Flatbuffers for go do not allow mutation (for complex types like strings). So we construct
	// a new flatbuffer entirely, copying the old one while replacing the entry we care to change.
	builder := flatbuffers.NewBuilder(0)
	users := d.copyUserAssociationList(builder)
	refs := d.copyReferenceListWithReplacement(
		builder,
		func(r *dscachefb.RefEntryInfo) bool {
			return string(r.InitID()) == act.InitID
		},
		func(refStartMutationFunc func(builder *flatbuffers.Builder)) {
			var metaTitle, commitTitle, commitMessage flatbuffers.UOffsetT
			if act.Dataset != nil && act.Dataset.Meta != nil {
				metaTitle = builder.CreateString(act.Dataset.Meta.Title)
			}
			if act.Dataset != nil && act.Dataset.Commit != nil {
				commitTitle = builder.CreateString(act.Dataset.Commit.Title)
				commitMessage = builder.CreateString(act.Dataset.Commit.Message)
			}
			hashRef := builder.CreateString(string(act.HeadRef))
			// Start building a ref object, by mutating an existing ref object.
			refStartMutationFunc(builder)
			// Add only the fields we want to change.
			dscachefb.RefEntryInfoAddTopIndex(builder, int32(act.TopIndex))
			dscachefb.RefEntryInfoAddCursorIndex(builder, int32(act.TopIndex))
			if act.Dataset != nil && act.Dataset.Meta != nil {
				dscachefb.RefEntryInfoAddMetaTitle(builder, metaTitle)
			}
			if act.Dataset != nil && act.Dataset.Commit != nil {
				dscachefb.RefEntryInfoAddCommitTime(builder, act.Dataset.Commit.Timestamp.Unix())
				dscachefb.RefEntryInfoAddCommitTitle(builder, commitTitle)
				dscachefb.RefEntryInfoAddCommitMessage(builder, commitMessage)
			}
			if act.Dataset != nil && act.Dataset.Structure != nil {
				dscachefb.RefEntryInfoAddBodySize(builder, int64(act.Dataset.Structure.Length))
				dscachefb.RefEntryInfoAddBodyRows(builder, int32(act.Dataset.Structure.Entries))
				dscachefb.RefEntryInfoAddNumErrors(builder, int32(act.Dataset.Structure.ErrCount))
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

func convertEntryToVersionInfo(r *dscachefb.RefEntryInfo) dsref.VersionInfo {
	return dsref.VersionInfo{
		InitID:        string(r.InitID()),
		ProfileID:     string(r.ProfileID()),
		Name:          string(r.PrettyName()),
		Path:          string(r.HeadRef()),
		Published:     r.Published(),
		Foreign:       r.Foreign(),
		MetaTitle:     string(r.MetaTitle()),
		ThemeList:     string(r.ThemeList()),
		BodySize:      int(r.BodySize()),
		BodyRows:      int(r.BodyRows()),
		BodyFormat:    string(r.BodyFormat()),
		NumErrors:     int(r.NumErrors()),
		CommitTime:    time.Unix(r.CommitTime(), 0),
		CommitTitle:   string(r.CommitTitle()),
		CommitMessage: string(r.CommitMessage()),
		NumVersions:   int(r.NumVersions()),
		FSIPath:       string(r.FsiPath()),
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
