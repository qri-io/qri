package cron

import (
	"fmt"
	"testing"

	flatbuffers "github.com/google/flatbuffers/go"
	cronfb "github.com/qri-io/qri/cron/cron_fbs"
)

func CompareJobSlices(a, b []*Job) error {
	if len(a) != len(b) {
		return fmt.Errorf("length mistmatch: %d != %d", len(a), len(b))
	}

	for i, jobA := range a {
		if err := CompareJobs(jobA, b[i]); err != nil {
			return fmt.Errorf("job index %d mistmatch: %s", i, err)
		}
	}

	return nil
}

func CompareJobs(a, b *Job) error {
	if a.Name != b.Name {
		return fmt.Errorf("Name mismatch. %s != %s", a.Name, b.Name)
	}
	if a.Periodicity != b.Periodicity {
		return fmt.Errorf("Periodicity mismatch. %s != %s", a.Name, b.Name)
	}
	// use unix comparisons to ignore millisecond & nanosecond precision errors
	if a.LastRunStart.Unix() != b.LastRunStart.Unix() {
		return fmt.Errorf("LastRunStart mismatch. %s != %s", a.LastRunStart, b.LastRunStart)
	}

	if a.Type != b.Type {
		return fmt.Errorf("Type mistmatch. %s != %s", a.Type, b.Type)
	}

	if err := CompareOptions(a.Options, b.Options); err != nil {
		return fmt.Errorf("Options: %s", err)
	}

	return nil
}

func CompareOptions(a, b Options) error {
	if a == nil && b != nil || a != nil && b == nil {
		return fmt.Errorf("nil mismatch: %v != %v", a, b)
	} else if a == nil && b == nil {
		return nil
	}

	aDso, aOk := a.(*DatasetOptions)
	bDso, bOk := b.(*DatasetOptions)
	if aOk && bOk {
		err := CompareDatasetOptions(aDso, bDso)
		if err != nil {
			return fmt.Errorf("DatasetOptions: %s", err)
		}
		return nil
	}

	// TODO (b5) - more option comparison
	return fmt.Errorf("TODO - can't compare option types: %#v %#v", a, b)
}

func TestDatasetOptionsFlatbuffer(t *testing.T) {
	src := &DatasetOptions{
		Title:     "A_Title",
		Message:   "A_Message",
		Recall:    "A_Recall",
		BodyPath:  "A_BodyPath",
		FilePaths: []string{"a", "b", "c"},

		Publish:             true,
		Strict:              true,
		Force:               true,
		ConvertFormatToPrev: true,
		ShouldRender:        true,

		Config:  map[string]string{"a": "a"},
		Secrets: map[string]string{"b": "b"},
	}

	builder := flatbuffers.NewBuilder(0)
	off := src.MarshalFlatbuffer(builder)
	if off == 0 {
		t.Errorf("expected returned offset to not equal zero")
	}
	builder.Finish(off)

	cronOpts := cronfb.GetRootAsDatasetOptions(builder.FinishedBytes(), 0)

	got := &DatasetOptions{}
	got.UnmarshalFlatbuffer(cronOpts)

	if err := CompareDatasetOptions(src, got); err != nil {
		t.Error(err)
	}
}

func CompareDatasetOptions(a, b *DatasetOptions) error {
	if a == nil && b != nil || a != nil && b == nil {
		return fmt.Errorf("nil mismatch: %v != %v", a, b)
	} else if a == nil && b == nil {
		return nil
	}

	if a.Title != b.Title {
		return fmt.Errorf("Title: '%s' != '%s'", a.Title, b.Title)
	}
	if a.Message != b.Message {
		return fmt.Errorf("Message: '%s' != '%s'", a.Message, b.Message)
	}
	if a.Recall != b.Recall {
		return fmt.Errorf("Recall: '%s' != '%s'", a.Recall, b.Recall)
	}
	if a.BodyPath != b.BodyPath {
		return fmt.Errorf("BodyPath: '%s' != '%s'", a.BodyPath, b.BodyPath)
	}

	if len(a.FilePaths) != len(b.FilePaths) {
		return fmt.Errorf("FilePaths length: %d != %d", len(a.FilePaths), len(b.FilePaths))
	}
	for i, ai := range a.FilePaths {
		if b.FilePaths[i] != ai {
			return fmt.Errorf("FilePaths index %d: %s != %s", i, ai, b.FilePaths[i])
		}
	}

	if a.Publish != b.Publish {
		return fmt.Errorf("Publish: %t != %t", a.Publish, b.Publish)
	}
	if a.Strict != b.Strict {
		return fmt.Errorf("Strict: %t != %t", a.Strict, b.Strict)
	}
	if a.Force != b.Force {
		return fmt.Errorf("ConvertFormatToPrev: %t != %t", a.Force, b.Force)
	}
	if a.ConvertFormatToPrev != b.ConvertFormatToPrev {
		return fmt.Errorf("ConvertFormatToPrev: %t != %t", a.ConvertFormatToPrev, b.ConvertFormatToPrev)
	}
	if a.ShouldRender != b.ShouldRender {
		return fmt.Errorf("ShouldRender: %t != %t", a.ShouldRender, b.ShouldRender)
	}

	if err := compareMapStringString(a.Config, b.Config); err != nil {
		return fmt.Errorf("Config: %s", err)
	}
	if err := compareMapStringString(a.Secrets, b.Secrets); err != nil {
		return fmt.Errorf("Secrets: %s", err)
	}

	return nil
}

func compareMapStringString(a, b map[string]string) error {
	if a == nil && b != nil || a != nil && b == nil {
		return fmt.Errorf("nil mismatch: %v != %v", a, b)
	} else if a == nil && b == nil {
		return nil
	}

	if len(a) != len(b) {
		return fmt.Errorf("length: %d != %d", len(a), len(b))
	}

	for k, v := range a {
		if v != b[k] {
			return fmt.Errorf("map key %s: %s != %s", k, v, b[k])
		}
	}

	return nil
}
