package cmd

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/update/cron"
)

type peerStringer config.ProfilePod

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

type refStringer repo.DatasetRef

// String assumes Peername and Name are present
func (r refStringer) String() string {
	w := &bytes.Buffer{}
	title := color.New(color.FgGreen, color.Bold).SprintFunc()
	path := color.New(color.Faint).SprintFunc()
	warn := color.New(color.FgYellow).SprintFunc()
	ds := r.Dataset
	dsr := repo.DatasetRef(r)

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

type logStringer repo.DatasetRef

// String assumes Path, Peername, Timestamp and Title are present
func (l logStringer) String() string {
	w := &bytes.Buffer{}
	// title := color.New(color.Bold).Sprintfunc()
	path := color.New(color.FgGreen).SprintFunc()
	dsr := repo.DatasetRef(l)

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

type jobStringer cron.Job

// String assumes Name, Type, Periodicity, and PrevRunStart are present
func (j jobStringer) String() string {
	w := &bytes.Buffer{}
	name := color.New(color.Bold).SprintFunc()
	t := j.Periodicity.After(j.PrevRunStart)
	relTime := humanize.RelTime(time.Now().In(time.UTC), t, "", "")
	fmt.Fprintf(w, "%s\nin %sat %s | %s\n", name(j.Name), relTime, t.In(time.Now().Location()).Format(time.Kitchen), j.Type)
	if j.RepoPath != "" {
		fmt.Fprintf(w, "\nrepo: %s\n", j.RepoPath)
	}
	fmt.Fprintf(w, "\n")
	return w.String()
}

type finishedJobStringer cron.Job

// String assumes Name, Type, PrevRunStart and ExitStatus are present
func (j finishedJobStringer) String() string {
	w := &bytes.Buffer{}
	name := color.New(color.Bold, color.FgGreen).SprintFunc()
	msg := ""
	if j.RunError != "" {
		msg = oneLiner(j.RunError, 40)
		name = color.New(color.Bold, color.FgRed).SprintFunc()
		if j.RunError == "no changes to save" {
			name = color.New(color.Bold, color.Faint).SprintFunc()
		}
	} else {
		if j.Type == cron.JTDataset {
			msg = "dataset updated"
		} else if j.Type == cron.JTShellScript {
			msg = "script ran successfully"
		}
	}

	fmt.Fprintf(w, "%s\n%s | %s\n", name(j.Name), humanize.Time(j.PrevRunStart), msg)
	if j.RepoPath != "" {
		fmt.Fprintf(w, "\nrepo: %s\n", j.RepoPath)
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
