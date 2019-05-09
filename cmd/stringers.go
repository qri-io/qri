package cmd

import (
	"bytes"
	"fmt"

	"github.com/fatih/color"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/repo"
)

type peerStringer config.ProfilePod

func (p peerStringer) String() string {
	w := &bytes.Buffer{}
	name := color.New(color.FgGreen, color.Bold).SprintFunc()
	online := color.New(color.FgYellow).SprintFunc()
	if p.Online {
		fmt.Fprintf(w, "%s | %s\n", name(p.Peername), online("online"))
	} else {
		fmt.Fprintf(w, "%s\n", name(p.Peername))
	}
	fmt.Fprintf(w, "profile ID: %s\n", p.ID)
	if len(p.NetworkAddrs) > 0 {
		fmt.Fprintf(w, "address:    %s\n", p.NetworkAddrs[0])
	}
	fmt.Fprintln(w, "")
	return w.String()
}

type stringer string

func (s stringer) String() string {
	return string(s) + "\n"
}

type refStringer repo.DatasetRef

func (r refStringer) String() string {
	w := &bytes.Buffer{}
	title := color.New(color.FgGreen, color.Bold).SprintFunc()
	path := color.New(color.Faint).SprintFunc()
	ds := r.Dataset
	dsr := repo.DatasetRef(r)

	fmt.Fprintf(w, "%s\n", title(dsr.AliasString()))
	if ds != nil && ds.Meta != nil && ds.Meta.Title != "" {
		fmt.Fprintf(w, "%s\n", ds.Meta.Title)
	}
	if r.Path != "" {
		fmt.Fprintf(w, "%s\n", path(r.Path))
	}
	if ds != nil && ds.Structure != nil {
		fmt.Fprintf(w, "%s", printByteInfo(ds.Structure.Length))
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

type logStringer repo.DatasetRef

func (r logStringer) String() string {
	w := &bytes.Buffer{}
	// title := color.New(color.Bold).Sprintfunc()
	path := color.New(color.FgGreen).SprintFunc()
	dsr := repo.DatasetRef(r)

	fmt.Fprintf(w, "%s%s\n", path("path:   "), path(dsr.Path))
	fmt.Fprintf(w, "Author: %s\n", dsr.Peername)
	fmt.Fprintf(w, "Date:   %s\n", dsr.Dataset.Commit.Timestamp.Format("Jan _2 15:04:05"))
	fmt.Fprintf(w, "\n\t%s\n", dsr.Dataset.Commit.Title)
	if dsr.Dataset.Commit.Message != "" {
		fmt.Fprintf(w, "\t%s\n", dsr.Dataset.Commit.Message)
	}

	fmt.Fprintf(w, "\n")
	return w.String()
}
