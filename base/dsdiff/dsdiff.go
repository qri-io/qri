package dsdiff

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/qri-io/dataset"
	jdiff "github.com/yudai/gojsondiff"
	"github.com/yudai/gojsondiff/formatter"
)

// SubDiff holds the diffs of a Dataset Subcomponent diff
type SubDiff struct {
	jdiff.Diff
	kind string
	a, b []byte
}

// MarshalJSON marshals the slice of diffs from the SubDiff
func (d *SubDiff) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.Diff.Deltas())
}

// SummarizeToString outputs a substring in a one of a few formats
// - simple (single line describing which component and how many
//   changes)
// - listKeys (lists the keys of what changed)
// - plusMinusColor (git-style plus/minus printout)
// - plusMinus (same as plusMinusColor without color)
func (d *SubDiff) SummarizeToString(how string) (string, error) {
	color := false
	if strings.Contains(how, "Color") {
		color = true
	}
	pluralS := ""
	if d != nil && d.Deltas() != nil && len(d.Deltas()) > 1 {
		pluralS = "s"
	}
	switch how {
	case "simple":
		if d.Modified() {
			componentTitle := strings.Title(d.kind)
			return fmt.Sprintf("%s Changed. (%d change%s)", componentTitle, len(d.Deltas()), pluralS), nil
		}
	case "listKeys":
		if d.Modified() {
			componentTitle := strings.Title(d.kind)
			namedDiffs := ""
			for _, del := range d.Deltas() {
				namedDiffs = fmt.Sprintf("%s\n\t- modified %s", namedDiffs, del)
			}
			return fmt.Sprintf("%s: %d change%s%s", componentTitle, len(d.Deltas()), pluralS, namedDiffs), nil
		}
	case "plusMinusColor", "plusMinus":
		if d.Modified() {
			var aJSON map[string]interface{}
			err := json.Unmarshal(d.a, &aJSON)
			if err != nil {
				return "", fmt.Errorf("error summarizing: %s", err.Error())
			}
			config := formatter.AsciiFormatterConfig{
				ShowArrayIndex: true,
				Coloring:       color,
			}
			form := formatter.NewAsciiFormatter(aJSON, config)
			diffString, err := form.Format(d)
			if err != nil {
				return "", fmt.Errorf("error summarizing: %s", err.Error())
			}
			return diffString, nil
		}
	case "delta":
		if d.Modified() {
			form := formatter.NewDeltaFormatter()
			diffString, err := form.Format(d)
			if err != nil {
				return "", fmt.Errorf("error summarizing: %s", err.Error())
			}
			return diffString, nil
		}
	default:
		return "", nil
	}
	return "", nil
}

// DiffStructure diffs the structure of two datasets
func DiffStructure(a, b *dataset.Structure) (*SubDiff, error) {
	var emptyDiff = &SubDiff{kind: "structure"}

	if a == nil {
		a = &dataset.Structure{}
	}

	if b == nil {
		b = &dataset.Structure{}
	}

	aBytes, err := a.MarshalJSONObject()
	if err != nil {
		return nil, fmt.Errorf("error marshalling structure a: %s", err.Error())
	}
	bBytes, err := b.MarshalJSONObject()
	if err != nil {
		return nil, fmt.Errorf("error marshalling structure b: %s", err.Error())
	}
	return DiffJSON(aBytes, bBytes, emptyDiff.kind)
}

// DiffData diffs the data of two datasets
func DiffData(a, b *dataset.Dataset) (*SubDiff, error) {
	var emptyDiff = &SubDiff{kind: "data"}
	// differ := jdiff.New()
	if len(a.BodyPath) > 1 && len(b.BodyPath) > 1 {
		if a.BodyPath == b.BodyPath {
			return emptyDiff, nil
		}
	}
	// TODO: dereference BodyPath and pass to jsondiffer
	return emptyDiff, nil
}

// DiffTransform diffs the transform struct of two datasets
func DiffTransform(a, b *dataset.Transform) (*SubDiff, error) {
	var emptyDiff = &SubDiff{kind: "transform"}

	// Make copies of the input structs, since we might modify them.
	var left, right dataset.Transform
	if a != nil {
		left = *a
	}
	if b != nil {
		right = *b
	}
	// The scriptPath is "incidental", and should not contribute to "signficant" differences.
	// As long as it is non-empty, copy it from one struct to the other.
	if left.ScriptPath != "" && right.ScriptPath != "" {
		right.ScriptPath = left.ScriptPath
	}

	if len(left.Path) > 1 && len(right.Path) > 1 {
		if left.Path == right.Path {
			return emptyDiff, nil
		}
	}
	leftBytes, err := left.MarshalJSONObject()
	if err != nil {
		return nil, fmt.Errorf("error marshalling transform a: %s", err.Error())
	}
	rightBytes, err := right.MarshalJSONObject()
	if err != nil {
		return nil, fmt.Errorf("error marshalling transform b: %s", err.Error())
	}
	return DiffJSON(leftBytes, rightBytes, emptyDiff.kind)
}

// DiffMeta diffs the metadata of two datasets
func DiffMeta(a, b *dataset.Meta) (*SubDiff, error) {
	var emptyDiff = &SubDiff{kind: "meta"}

	if a == nil {
		a = &dataset.Meta{}
	}
	if b == nil {
		b = &dataset.Meta{}
	}

	if len(a.Path) > 1 && len(b.Path) > 1 {
		if a.Path == b.Path {
			return emptyDiff, nil
		}
	} else if a.IsEmpty() && b.IsEmpty() {
		return emptyDiff, nil
	}
	aBytes, err := a.MarshalJSONObject()
	if err != nil {
		return nil, fmt.Errorf("error marshalling meta a: %s", err.Error())
	}
	bBytes, err := b.MarshalJSONObject()
	if err != nil {
		return nil, fmt.Errorf("error marshalling meta b: %s", err.Error())
	}
	return DiffJSON(aBytes, bBytes, emptyDiff.kind)
}

// DiffViz diffs the dataset.Viz structs of two datasets
func DiffViz(a, b *dataset.Viz) (*SubDiff, error) {
	var emptyDiff = &SubDiff{kind: "viz"}

	if a == nil {
		a = &dataset.Viz{}
	}
	if b == nil {
		b = &dataset.Viz{}
	}

	if len(a.Path) > 1 && len(b.Path) > 1 {
		if a.Path == b.Path {
			return emptyDiff, nil
		}
	}
	aBytes, err := a.MarshalJSONObject()
	if err != nil {
		return nil, fmt.Errorf("error marshalling viz a: %s", err.Error())
	}
	bBytes, err := b.MarshalJSONObject()
	if err != nil {
		return nil, fmt.Errorf("error marshalling viz b: %s", err.Error())
	}
	return DiffJSON(aBytes, bBytes, emptyDiff.kind)
}

//DiffJSON diffs two json byte slices and returns a SubDiff pointer
func DiffJSON(a, b []byte, kind string) (*SubDiff, error) {
	differ := jdiff.New()
	d, err := differ.Compare(a, b)
	if err != nil {
		// return emptyDiff, fmt.Errorf("error comparing %s: %s", kind, err.Error())
		return nil, fmt.Errorf("error comparing %s: %s", kind, err.Error())
	}
	subDiff := &SubDiff{d, kind, a, b}
	return subDiff, nil
}

// StructuredDataTuple provides an additional input for DiffDatasets
// to use fully de-referenced dataset.data so that we can consider
// changes in dataset.Data beyond the hash/path being similar or
// different
type StructuredDataTuple struct {
	a, b *[]byte
}

// DiffDatasets returns a map of pointers to diffs of the components
// of a dataset.  It calls each of the Diff{Component} functions and
// adds the option for including de-referenced dataset.Data via
// the StructuredDataTuple
// TODO (kasey): This function only diffs: Structure, Meta, Transform, Viz, and Data (in JSON or as a Dataset)
func DiffDatasets(a, b *dataset.Dataset, deRefData *StructuredDataTuple) (map[string]*SubDiff, error) {
	result := make(map[string]*SubDiff)

	//diff structure
	if a.Structure != nil || b.Structure != nil {
		structureDiffs, err := DiffStructure(a.Structure, b.Structure)
		if err != nil {
			return result, err
		}
		if structureDiffs.Diff != nil {
			result[structureDiffs.kind] = structureDiffs
		}
	}
	// diff data
	if deRefData != nil {
		dataDiffs, err := DiffJSON(*deRefData.a, *deRefData.b, "data")
		if err != nil {
			return nil, err
		}
		result[dataDiffs.kind] = dataDiffs
	} else {
		dataDiffs, err := DiffData(a, b)
		if err != nil {
			return nil, err
		}
		if dataDiffs.Diff != nil {
			result[dataDiffs.kind] = dataDiffs
		}
	}
	// diff meta
	if a.Meta != nil || b.Meta != nil {
		metaDiffs, err := DiffMeta(a.Meta, b.Meta)
		if err != nil {
			return nil, err
		}
		if metaDiffs.Diff != nil {
			result[metaDiffs.kind] = metaDiffs
		}
	}
	// diff transform
	if a.Transform != nil || b.Transform != nil {
		transformDiffs, err := DiffTransform(a.Transform, b.Transform)
		if err != nil {
			return nil, err
		}
		if transformDiffs.Diff != nil {
			result[transformDiffs.kind] = transformDiffs
		}
	}
	// diff viz
	if a.Viz != nil || b.Viz != nil {
		vizDiffs, err := DiffViz(a.Viz, b.Viz)
		if err != nil {
			return nil, err
		}
		if vizDiffs.Diff != nil {
			result[vizDiffs.kind] = vizDiffs
		}
	}
	return result, nil
}

// MapDiffsToString generates a string description from a map of diffs
// Currently the String generated reflects the first/highest priority
// change made.  The priority of changes currently are
//   1. dataset.Structure
//   2. dataset.{Data}
//   3. dataset.Transform
//   4. dataset.Meta
//   5. Dataset.Viz
func MapDiffsToString(m map[string]*SubDiff, how string) (string, error) {
	keys := []string{
		"structure",
		"data",
		"transform",
		"meta",
		"viz",
	}
	// for _, key := range keys {
	// 	val, ok := m[key]
	// 	fmt.Printf("%s: %s, %t\n===\n", key, val, ok)
	// }
	for _, key := range keys {
		diffs, ok := m[key]
		if ok && diffs != nil {
			summary, err := diffs.SummarizeToString(how)
			if err != nil {
				return "", fmt.Errorf("error summarizing %s: %s", diffs.kind, err.Error())
			}
			if summary != "" {
				return summary, nil
			}
		}
	}
	return "", nil
}
