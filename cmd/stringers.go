package cmd

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/lib"
	reporef "github.com/qri-io/qri/repo/ref"
)

type peerStringer config.ProfilePod

// StringerLocation is the function to retrieve the timezone location
var StringerLocation *time.Location

func init() {
	StringerLocation = time.Now().Location()
}

// String assumes that Peername and ID are present
func (p peerStringer) String() string {
	w := &bytes.Buffer{}
	name := color.New(color.FgGreen, color.Bold).SprintFunc()
	online := color.New(color.FgYellow).SprintFunc()
	if p.Online {
		fmt.Fprintf(w, "%s | %s\n", name(p.Peername), online("online"))
	} else {
		fmt.Fprintf(w, "%s\n", name(p.Peername))
	}
	fmt.Fprintf(w, "Profile ID: %s\n", p.ID)
	plural := "es"
	spacer := "              "
	if len(p.NetworkAddrs) <= 1 {
		plural = ""
	}
	for i, addr := range p.NetworkAddrs {
		if i == 0 {
			fmt.Fprintf(w, "Address%s:    %s\n", plural, addr)
			continue
		}
		fmt.Fprintf(w, "%s%s\n", spacer, addr)
	}
	fmt.Fprintln(w, "")
	return w.String()
}

type stringer string

func (s stringer) String() string {
	return string(s) + "\n"
}

type refStringer reporef.DatasetRef

// String assumes Peername and Name are present
func (r refStringer) String() string {
	w := &bytes.Buffer{}
	title := color.New(color.FgGreen, color.Bold).SprintFunc()
	path := color.New(color.Faint).SprintFunc()
	warn := color.New(color.FgYellow).SprintFunc()
	ds := r.Dataset
	dsr := reporef.DatasetRef(r)

	fmt.Fprintf(w, "%s", title(dsr.AliasString()))
	if ds != nil && ds.Meta != nil && ds.Meta.Title != "" {
		fmt.Fprintf(w, "\n%s", ds.Meta.Title)
	}
	if r.FSIPath != "" {
		fmt.Fprintf(w, "\nlinked: %s", path(r.FSIPath))
	} else if r.Path != "" {
		fmt.Fprintf(w, "\n%s", path(r.Path))
	}
	if r.Foreign {
		fmt.Fprintf(w, "\n%s", warn("foreign"))
	}
	if ds != nil && ds.Structure != nil {
		fmt.Fprintf(w, "\n%s", humanize.Bytes(uint64(ds.Structure.Length)))
		if ds.Structure.Entries == 1 {
			fmt.Fprintf(w, ", %d entry", ds.Structure.Entries)
		} else {
			fmt.Fprintf(w, ", %d entries", ds.Structure.Entries)
		}
		if ds.Structure.ErrCount == 1 {
			fmt.Fprintf(w, ", %d error", ds.Structure.ErrCount)
		} else {
			fmt.Fprintf(w, ", %d errors", ds.Structure.ErrCount)
		}
		if ds.NumVersions == 0 {
			// nothing
		} else if ds.NumVersions == 1 {
			fmt.Fprintf(w, ", %d version", ds.NumVersions)
		} else {
			fmt.Fprintf(w, ", %d versions", ds.NumVersions)
		}
	}

	fmt.Fprintf(w, "\n\n")
	return w.String()
}

type versionInfoStringer dsref.VersionInfo

// String assumes Peername and Name are present
func (vis versionInfoStringer) String() string {
	w := &bytes.Buffer{}
	title := color.New(color.FgGreen, color.Bold).SprintFunc()
	path := color.New(color.Faint).SprintFunc()
	warn := color.New(color.FgYellow).SprintFunc()

	v := dsref.VersionInfo(vis)
	sr := v.SimpleRef()
	fmt.Fprintf(w, "%s", title(sr.Alias()))

	if vis.MetaTitle != "" {
		fmt.Fprintf(w, "\n%s", vis.MetaTitle)
	}
	if vis.FSIPath != "" {
		fmt.Fprintf(w, "\nlinked: %s", path(vis.FSIPath))
	} else if vis.Path != "" {
		fmt.Fprintf(w, "\n%s", path(vis.Path))
	}
	if vis.Foreign {
		fmt.Fprintf(w, "\n%s", warn("foreign"))
	}
	fmt.Fprintf(w, "\n%s", humanize.Bytes(uint64(vis.BodySize)))
	if vis.BodyRows == 1 {
		fmt.Fprintf(w, ", %d entry", vis.BodyRows)
	} else {
		fmt.Fprintf(w, ", %d entries", vis.BodyRows)
	}
	if vis.NumErrors == 1 {
		fmt.Fprintf(w, ", %d error", vis.NumErrors)
	} else {
		fmt.Fprintf(w, ", %d errors", vis.NumErrors)
	}
	if vis.NumVersions == 0 {
		// nothing
	} else if vis.NumVersions == 1 {
		fmt.Fprintf(w, ", %d version", vis.NumVersions)
	} else {
		fmt.Fprintf(w, ", %d versions", vis.NumVersions)
	}

	fmt.Fprintf(w, "\n\n")
	return w.String()
}

type searchResultStringer lib.SearchResult

func (r searchResultStringer) String() string {
	w := &strings.Builder{}
	title := color.New(color.FgGreen, color.Bold).SprintFunc()
	path := color.New(color.Faint).SprintFunc()
	ds := r.Value

	fmt.Fprintf(w, "%s/%s", title(ds.Peername), title(ds.Name))
	fmt.Fprintf(w, "\n%s", r.URL)
	fmt.Fprintf(w, "\n%s", path(ds.Path))

	if ds != nil && ds.Meta != nil && ds.Meta.Title != "" {
		fmt.Fprintf(w, "\n%s", ds.Meta.Title)
	}
	if ds != nil && ds.Structure != nil {
		fmt.Fprintf(w, "\n%s", humanize.Bytes(uint64(ds.Structure.Length)))
		if ds.Structure.Entries == 1 {
			fmt.Fprintf(w, ", %d entry", ds.Structure.Entries)
		} else {
			fmt.Fprintf(w, ", %d entries", ds.Structure.Entries)
		}
		if ds.Structure.ErrCount == 1 {
			fmt.Fprintf(w, ", %d error", ds.Structure.ErrCount)
		} else {
			fmt.Fprintf(w, ", %d errors", ds.Structure.ErrCount)
		}
		if ds.NumVersions == 0 {
			// nothing
		} else if ds.NumVersions == 1 {
			fmt.Fprintf(w, ", %d version", ds.NumVersions)
		} else {
			fmt.Fprintf(w, ", %d versions", ds.NumVersions)
		}
	}

	fmt.Fprintf(w, "\n\n")
	return w.String()
}

type logStringer reporef.DatasetRef

// String assumes Path, Peername, Timestamp and Title are present
func (l logStringer) String() string {
	w := &bytes.Buffer{}
	// title := color.New(color.Bold).Sprintfunc()
	path := color.New(color.FgGreen).SprintFunc()
	dsr := reporef.DatasetRef(l)

	fmt.Fprintf(w, "%s\n", path("path:   "+dsr.Path))
	fmt.Fprintf(w, "Author: %s\n", dsr.Peername)
	fmt.Fprintf(w, "Date:   %s\n", dsr.Dataset.Commit.Timestamp.Format("Jan _2 15:04:05"))
	fmt.Fprintf(w, "\n    %s\n", dsr.Dataset.Commit.Title)
	if dsr.Dataset.Commit.Message != "" {
		fmt.Fprintf(w, "    %s\n", dsr.Dataset.Commit.Message)
	}

	fmt.Fprintf(w, "\n")
	return w.String()
}

func oneLiner(str string, maxLen int) string {
	str = strings.Split(str, "\n")[0]
	if len(str) > maxLen-3 {
		str = str[:maxLen-3] + "..."
	}
	return str
}

type logEntryStringer lib.LogEntry

func (s logEntryStringer) String() string {
	title := color.New(color.FgGreen, color.Bold).SprintFunc()
	ts := color.New(color.Faint).SprintFunc()

	return fmt.Sprintf("%s\t%s\t%s\t%s\n",
		ts(s.Timestamp.Format(time.RFC3339)),
		title(s.Author),
		title(s.Action),
		s.Note,
	)
}

type dslogItemStringer DatasetLogItem

func (s dslogItemStringer) String() string {
	yellow := color.New(color.FgYellow).SprintFunc()
	faint := color.New(color.Faint).SprintFunc()

	storage := "local"
	if s.Foreign {
		storage = faint("remote")
	}

	msg := fmt.Sprintf("%s%s\n%s%s\n%s%s\n%s%s\n\n%s\n",
		faint("Commit:  "),
		yellow(s.Path),
		faint("Date:    "),
		s.CommitTime.In(StringerLocation).Format(time.UnixDate),
		faint("Storage: "),
		storage,
		faint("Size:    "),
		humanize.Bytes(uint64(s.BodySize)),
		s.CommitTitle,
	)
	if s.CommitMessage != "" && s.CommitMessage != s.CommitTitle {
		msg += fmt.Sprintf("%s\n", s.CommitMessage)
	}
	msg += "\n"

	return msg
}
