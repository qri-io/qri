package friendly

import (
	"fmt"
	"strings"

	golog "github.com/ipfs/go-log"
	"github.com/qri-io/deepdiff"
)

var log = golog.Logger("friendly")

const smallNumberOfChangesToBody = 3

// ComponentChanges holds state when building a diff message
type ComponentChanges struct {
	EntireMessage string
	Num           int
	Size          int
	Rows          []string
}

// DiffDescriptions creates a friendly message from diff operations. If there's no differences
// found, return empty strings.
func DiffDescriptions(headDeltas, bodyDeltas []*deepdiff.Delta, bodyStats *deepdiff.Stats, assumeBodyChanged bool) (string, string) {
	log.Debugw("DiffDescriptions", "len(headDeltas)", len(headDeltas), "len(bodyDeltas)", len(bodyDeltas), "bodyStats", bodyStats, "assumeBodyChanged", assumeBodyChanged)
	if len(headDeltas) == 0 && len(bodyDeltas) == 0 {
		return "", ""
	}

	headDeltas = preprocess(headDeltas, "")
	bodyDeltas = preprocess(bodyDeltas, "")

	perComponentChanges := buildComponentChanges(headDeltas, bodyDeltas, bodyStats, assumeBodyChanged)

	// Data accumulated while iterating over the components.
	shortTitle := ""
	longMessage := ""
	changedComponents := []string{}

	// Iterate over certain components that we want to check for changes to.
	componentsToCheck := []string{"meta", "structure", "readme", "viz", "transform", "body"}
	for _, compName := range componentsToCheck {
		if changes, ok := perComponentChanges[compName]; ok {
			log.Debugw("checking component", "name", compName)
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
					// Take the max of left and right to calculate the percentage change.
					divisor := bodyStats.Left
					if bodyStats.Right > divisor {
						divisor = bodyStats.Right
					}
					percentChange := int(100.0 * changes.Size / divisor)
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
				if len(changes.Rows) == 0 {
					// This should never happen. If a component has no changes, it should not have
					// a key in the perComponentChanges map.
					log.Errorf("for %s: changes.Row is zero-sized", compName)
				} else if len(changes.Rows) == 1 {
					// For any other component, if there's only one change, directly describe
					// it for both the long message and short title.
					msg = fmt.Sprintf("%s:\n\t%s", compName, changes.Rows[0])
					shortTitle = fmt.Sprintf("%s %s", compName, changes.Rows[0])
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
		build = append(build, d)
		if len(d.Deltas) > 0 {
			d.Deltas = preprocess(d.Deltas, joinPath(path, d.Path.String()))
		}
	}
	return build
}

func buildComponentChanges(headDeltas, bodyDeltas deepdiff.Deltas, bodyStats *deepdiff.Stats, assumeBodyChanged bool) map[string]*ComponentChanges {
	perComponentChanges := make(map[string]*ComponentChanges)
	for _, d := range headDeltas {
		compName := d.Path.String()
		if d.Type != deepdiff.DTContext {
			// Entire component changed
			if d.Type == deepdiff.DTInsert || d.Type == deepdiff.DTDelete || d.Type == dtReplace {
				if _, ok := perComponentChanges[compName]; !ok {
					perComponentChanges[compName] = &ComponentChanges{}
				}
				changes := perComponentChanges[compName]
				changes.EntireMessage = pastTense(string(d.Type))
				continue
			} else {
				log.Debugf("unknown delta type %q for path %q", d.Type, d.Path)
				continue
			}
		} else if len(d.Deltas) > 0 {
			// Part of the component changed, record some state to build into a message later
			changes := &ComponentChanges{}
			buildChanges(changes, "", d.Deltas)
			perComponentChanges[compName] = changes
		}
	}
	if assumeBodyChanged {
		perComponentChanges["body"] = &ComponentChanges{EntireMessage: "changed"}
	} else if len(bodyDeltas) > 0 && bodyStats != nil {
		bodyChanges := &ComponentChanges{}
		buildBodyChanges(bodyChanges, "", bodyDeltas)
		if bodyChanges.Num > 0 {
			perComponentChanges["body"] = bodyChanges
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
			if valArray, ok := d.Value.([]interface{}); ok {
				changes.Size += len(valArray) + 1
			} else if valMap, ok := d.Value.(map[string]interface{}); ok {
				changes.Size += len(valMap) + 1
			} else {
				changes.Size++
			}
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
