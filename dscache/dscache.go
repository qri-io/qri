package dscache

import (
	"fmt"
	"io/ioutil"
	"strings"

	golog "github.com/ipfs/go-log"
	"github.com/qri-io/dataset"
	dscachefb "github.com/qri-io/qri/dscache/dscachefb"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
)

var (
	log = golog.Logger("dscache")
)

// Dscache represents an in-memory serialized dscache flatbuffer
type Dscache struct {
	Root                *dscachefb.Dscache
	Buffer              []byte
	ProfileIDToUsername map[string]string
}

// LoadDscacheFromFile will load a dscache from the given filename
func LoadDscacheFromFile(filename string) (*Dscache, error) {
	buffer, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	root := dscachefb.GetRootAsDscache(buffer, 0)
	return &Dscache{Root: root, Buffer: buffer}, nil
}

// SaveTo writes the serialized bytes to the given filename
func (d *Dscache) SaveTo(filename string) error {
	return ioutil.WriteFile(filename, d.Buffer, 0644)
}

// VerboseString is a convenience function that returns a readable string, for testing and debugging
func (d *Dscache) VerboseString(showEmpty bool) string {
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
		r := dscachefb.RefCache{}
		d.Root.Refs(&r, i)
		fmt.Fprintf(&out, ` %2d) initID      = %s
     profileID   = %s
     topIndex    = %d
     cursorIndex = %d
     prettyName  = %s
`, i, r.InitID(), r.ProfileID(), r.TopIndex(), r.CursorIndex(), r.PrettyName())
		indent := "     "
		if len(r.MetaTitle()) != 0 || showEmpty {
			fmt.Fprintf(&out, "%smetaTitle   = %s\n", indent, r.MetaTitle())
		}
		if len(r.ThemeList()) != 0 || showEmpty {
			fmt.Fprintf(&out, "%sthemeList   = %s\n", indent, r.ThemeList())
		}
		if r.BodySize() != 0 || showEmpty {
			fmt.Fprintf(&out, "%sbodySize    = %d\n", indent, r.BodySize())
		}
		if r.BodyRows() != 0 || showEmpty {
			fmt.Fprintf(&out, "%sbodyRows    = %d\n", indent, r.BodyRows())
		}
		if r.CommitTime() != 0 || showEmpty {
			fmt.Fprintf(&out, "%scommitTime  = %d\n", indent, r.CommitTime())
		}
		if r.NumErrors() != 0 || showEmpty {
			fmt.Fprintf(&out, "%snumErrors   = %d\n", indent, r.NumErrors())
		}
		if len(r.HeadRef()) != 0 || showEmpty {
			fmt.Fprintf(&out, "%sheadRef     = %s\n", indent, r.HeadRef())
		}
		if len(r.FsiPath()) != 0 || showEmpty {
			fmt.Fprintf(&out, "%sfsiPath     = %s\n", indent, r.FsiPath())
		}
	}
	return out.String()
}

// ListRefs returns references to each dataset in the cache
// TODO(dlong): Not alphabetized, which lib assumes it is
func (d *Dscache) ListRefs() ([]repo.DatasetRef, error) {
	d.ensureProToUserMap()
	refs := make([]repo.DatasetRef, 0, d.Root.RefsLength())
	for i := 0; i < d.Root.RefsLength(); i++ {
		refCache := dscachefb.RefCache{}
		d.Root.Refs(&refCache, i)
		profileID, err := profile.NewB58ID(string(refCache.ProfileID()))
		if err != nil {
			log.Errorf("could not parse profileID %q", string(refCache.ProfileID()))
			continue
		}

		refs = append(refs, repo.DatasetRef{
			Peername:  d.ProfileIDToUsername[string(refCache.ProfileID())],
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
