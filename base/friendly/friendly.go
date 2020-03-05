package friendly

import (
	"fmt"
	"strings"

	logger "github.com/ipfs/go-log"
	"github.com/qri-io/deepdiff"
)

var log = logger.Logger("friendly")

const smallNumberOfChangesToBody = 3

// ComponentChanges holds state when building a diff message
type ComponentChanges struct {
	EntireMessage string
	Num           int
	Rows          []string
}

// DiffDescriptions creates a friendly message from a diff operation
func DiffDescriptions(deltas []*deepdiff.Delta, stats *deepdiff.Stats) (string, string) {
	if len(deltas) == 0 {
		return "", ""
	}

	deltas = preprocess(deltas, "")
	perComponentChanges := buildComponentChanges(deltas)

	// Data accumulated while iterating over the components.
	shortTitle := ""
	longMessage := ""
	changedComponents := []string{}

	// Iterate over certain components that we want to check for changes to.
	componentsToCheck := []string{"meta", "structure", "readme", "viz", "transform", "body"}
	for _, compName := range componentsToCheck {
		if changes, ok := perComponentChanges[compName]; ok {
			changedComponents = append(changedComponents, compName)

			// Decide heuristically which type of message to use for this component
			var msg string
			if changes.EntireMessage != "" {
				// If there's a single message that describes the change for this component,
				// use that. Currently only used for deletes.
				msg = fmt.Sprintf("%s %s", compName, changes.EntireMessage)
				shortTitle = msg
			} else if compName == "body" {
				if changes.Rows == nil {
					// Body works specially. If a significant number of changes have been made,
					// just report the percentage of the body that has changed.
					percentChange := int(100.0 * changes.Num / stats.Left)
					action := fmt.Sprintf("changed by %d%%", percentChange)
					msg = fmt.Sprintf("%s:\n\t%s", compName, action)
					shortTitle = fmt.Sprintf("%s %s", compName, action)
				} else {
					// If only a small number of changes were made, then describe each of them.
					msg = fmt.Sprintf("%s:", compName)
					for _, r := range changes.Rows {
						msg = fmt.Sprintf("%s\n\t%s", msg, r)
					}
					shortTitle = fmt.Sprintf("%s %s", compName, strings.Join(changes.Rows, " and "))
				}
			} else {
				if len(changes.Rows) == 1 {
					// For any other component, if there's only one change, directly describe
					// it for both the long message and short title.
					msg = fmt.Sprintf("%s:\n\t%s", compName, changes.Rows[0])
					shortTitle = fmt.Sprintf("%s %s", compName, changes.Rows[0])
				} else if len(changes.Rows) == 0 && compName == "transform" {
					// TODO (b5) - this is a hack to make TestSaveTransformWithoutChanges
					// in the cmd package pass. We're getting to this stage with 0 rows of
					// changes, which is making msg & title not-empty, which is in turn allowing
					// a commit b/c it looks like a change. ideally we don't make it here at all,
					// but we DEFINITELY need a better hueristic for dsfs.CreateDataset's
					// change detection check. Maybe use diffstat?
					continue
				} else {
					// If there were multiple changes, describe them all for the long message
					// but just show the number of changes for the short title.
					msg = fmt.Sprintf("%s:", compName)
					for _, r := range changes.Rows {
						msg = fmt.Sprintf("%s\n\t%s", msg, r)
					}
					shortTitle = fmt.Sprintf("%s updated %d fields", compName, len(changes.Rows))
				}
			}
			// Append to full long message
			if longMessage == "" {
				longMessage = msg
			} else {
				longMessage = fmt.Sprintf("%s\n%s", longMessage, msg)
			}
		}
	}

	// Check if there were 2 or more components that got changed. If so, the short title will
	// just list the names of those components, with no additional detail.
	if len(changedComponents) == 2 {
		shortTitle = fmt.Sprintf("updated %s and %s", changedComponents[0], changedComponents[1])
	} else if len(changedComponents) > 2 {
		text := "updated "
		for k, compName := range changedComponents {
			if k == len(changedComponents)-1 {
				// If last change in the list...
				text = fmt.Sprintf("%sand %s", text, compName)
			} else {
				text = fmt.Sprintf("%s%s, ", text, compName)
			}
		}
		shortTitle = text
	}

	return shortTitle, longMessage
}

const dtReplace = deepdiff.Operation("replace")

// preprocess makes delta lists easier to work with, by combining operations
// when possible & removing unwanted paths
func preprocess(deltas deepdiff.Deltas, path string) deepdiff.Deltas {
	build := make([]*deepdiff.Delta, 0, len(deltas))
	for i, d := range deltas {
		if i > 0 {
			last := build[len(build)-1]
			if last.Path.String() == d.Path.String() {
				if last.Type == deepdiff.DTDelete && d.Type == deepdiff.DTInsert {
					last.Type = dtReplace
					continue
				}
			}
		}

		p := joinPath(path, d.Path.String())
		// TODO (b5) - We need this because it's possible to write a transform script
		// that changes nothing other than the script itself, and we currently reject
		// that change. Should we? I think I'm missing something.
		if p == "transform.scriptPath" {
			continue
		}

		build = append(build, d)
		if len(d.Deltas) > 0 {
			d.Deltas = preprocess(d.Deltas, joinPath(path, d.Path.String()))
		}
	}
	return build
}

func buildComponentChanges(deltas deepdiff.Deltas) map[string]*ComponentChanges {
	perComponentChanges := make(map[string]*ComponentChanges)
	for _, d := range deltas {
		compName := d.Path.String()
		if d.Type != deepdiff.DTContext {
			// Entire component changed
			if d.Type == deepdiff.DTInsert || d.Type == deepdiff.DTDelete || d.Type == dtReplace {
				if _, ok := perComponentChanges[compName]; !ok {
					perComponentChanges[compName] = &ComponentChanges{}
				}
				changes, _ := perComponentChanges[compName]
				changes.EntireMessage = pastTense(string(d.Type))
				continue
			} else {
				log.Debugf("unknown delta type %q for path %q", d.Type, d.Path)
				continue
			}
		} else if len(d.Deltas) > 0 {
			// Part of the component changed, record some state to build into a message later
			changes := &ComponentChanges{}

			if compName == "body" {
				buildBodyChanges(changes, "", d.Deltas)
			} else {
				buildChanges(changes, "", d.Deltas)
			}

			perComponentChanges[compName] = changes
		}
	}
	return perComponentChanges
}

func buildChanges(changes *ComponentChanges, parentPath string, deltas deepdiff.Deltas) {
	for _, d := range deltas {
		if d.Type != deepdiff.DTContext {
			rowModify := fmt.Sprintf("%s %s", pastTense(string(d.Type)), joinPath(parentPath, d.Path.String()))
			changes.Rows = append(changes.Rows, rowModify)
		}

		if len(d.Deltas) > 0 {
			buildChanges(changes, joinPath(parentPath, d.Path.String()), d.Deltas)
		}
	}
}

func buildBodyChanges(changes *ComponentChanges, parentPath string, deltas deepdiff.Deltas) {
	for _, d := range deltas {
		if d.Type == deepdiff.DTDelete || d.Type == deepdiff.DTInsert || d.Type == deepdiff.DTUpdate || d.Type == dtReplace {
			changes.Num++
			if changes.Num <= smallNumberOfChangesToBody {
				rowModify := fmt.Sprintf("%s row %s", pastTense(string(d.Type)), joinPath(parentPath, d.Path.String()))
				changes.Rows = append(changes.Rows, rowModify)
			} else {
				changes.Rows = nil
			}
		} else if len(d.Deltas) > 0 {
			buildBodyChanges(changes, joinPath(parentPath, d.Path.String()), d.Deltas)
		}
	}
}

func joinPath(parent, element string) string {
	if parent == "" {
		return element
	}
	return fmt.Sprintf("%s.%s", parent, element)
}

func pastTense(text string) string {
	if text == string(deepdiff.DTDelete) {
		return "removed"
	} else if text == string(deepdiff.DTInsert) {
		return "added"
	} else if text == string(deepdiff.DTUpdate) {
		return "updated"
	} else if text == "replace" {
		return "updated"
	}
	return text
}
