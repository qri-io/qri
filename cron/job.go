package cron

import (
	"fmt"
	"time"

	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/qri-io/iso8601"
	cronfb "github.com/qri-io/qri/cron/cron_fbs"
)

// JobType is a type for distinguishing between two different kinds of jobs
// JobType should be used as a shorthand for defining how to execute a job
type JobType string

const (
	// JTDataset indicates a job that runs "qri update" on a dataset specified
	// by Job Name. The job periodicity is determined by the specified dataset's
	// Meta.AccrualPeriodicity field. LastRun should closely match the datasets's
	// latest Commit.Timestamp value
	JTDataset JobType = "dataset"
	// JTShellScript represents a shell script to be run locally, which might
	// update one or more datasets. A non-zero exit code from shell script
	// indicates the job failed to execute properly
	JTShellScript JobType = "shell"
)

// Enum returns the enumerated representation of a JobType
func (jt JobType) Enum() int8 {
	switch jt {
	case JTDataset:
		return 1
	case JTShellScript:
		return 2
	}
	// "unknown"
	return 0
}

// Job represents a "cron job" that can be scheduled for repeated execution at
// a specified Periodicity (time interval)
type Job struct {
	Name        string
	Type        JobType
	Periodicity iso8601.RepeatingInterval

	LastRun   time.Time
	LastError string

	Options Options
}

// Validate confirms a Job contains valid details for scheduling
func (job *Job) Validate() error {
	if job.Name == "" {
		return fmt.Errorf("name is required")
	}
	zero := iso8601.RepeatingInterval{}
	if job.Periodicity == zero {
		return fmt.Errorf("period is required")
	}
	if job.Type != JTDataset && job.Type != JTShellScript {
		return fmt.Errorf("invalid job type: %s", job.Type)
	}
	return nil
}

// NextExec returns the next time execution horizion. If job periodicity is
// improperly configured, the returned time will be zero
func (job *Job) NextExec() time.Time {
	return job.Periodicity.After(job.LastRun)
}

// MarshalFlatbuffer writes a job to a builder
func (job *Job) MarshalFlatbuffer(builder *flatbuffers.Builder) flatbuffers.UOffsetT {
	name := builder.CreateString(job.Name)
	// typ := builder.CreateString(string(job.Type))

	lastRun := builder.CreateString(job.LastRun.Format(time.RFC3339))
	lastError := builder.CreateString(job.LastError)
	p := builder.CreateString(job.Periodicity.String())

	var opts flatbuffers.UOffsetT
	if job.Options != nil {
		opts = job.Options.MarshalFlatbuffer(builder)
	}

	cronfb.JobStart(builder)
	cronfb.JobAddName(builder, name)
	cronfb.JobAddType(builder, job.Type.Enum())
	cronfb.JobAddLastRun(builder, lastRun)
	cronfb.JobAddLastError(builder, lastError)
	cronfb.JobAddPeriodicity(builder, p)
	if opts != 0 {
		cronfb.JobAddOptions(builder, opts)
	}
	return cronfb.JobEnd(builder)
}

// UnmarshalFlatbuffer decodes a job from a flatbuffer
func (job *Job) UnmarshalFlatbuffer(j *cronfb.Job) error {
	lastRun, err := time.Parse(time.RFC3339, string(j.LastRun()))
	if err != nil {
		return err
	}

	p, err := iso8601.ParseRepeatingInterval(string(j.Periodicity()))
	if err != nil {
		return err
	}

	*job = Job{
		Name:        string(j.Name()),
		Type:        JobType(cronfb.EnumNamesJobType[j.Type()]),
		Periodicity: p,

		LastRun:   lastRun,
		LastError: string(j.LastError()),

		// TODO (b5) - deserializeOptions
	}
	return nil
}

// Options is an interface for job options
type Options interface {
	MarshalFlatbuffer(builder *flatbuffers.Builder) flatbuffers.UOffsetT
}

// ShellScriptOptions encapsulates options for running a shell script cron job
type ShellScriptOptions struct {
	// none yet.
}

// MarshalFlatbuffer writes to a builder
func (o *ShellScriptOptions) MarshalFlatbuffer(builder *flatbuffers.Builder) flatbuffers.UOffsetT {
	return 0
}

// DatasetOptions encapsulates options passed to `qri save`
type DatasetOptions struct {
	CommitTitle   string
	CommitMessage string
	Recall        string
	BodyPath      string
	FilePaths     []string

	Publish             bool
	Strict              bool
	Force               bool
	ConvertFormatToPrev bool
	NoRender            bool

	Config  map[string]string
	Secrets map[string]string
}

// MarshalFlatbuffer writes to a builder
func (o *DatasetOptions) MarshalFlatbuffer(builder *flatbuffers.Builder) flatbuffers.UOffsetT {
	commitTitle := builder.CreateString(o.CommitTitle)
	commitMessage := builder.CreateString(o.CommitMessage)
	recall := builder.CreateString(o.Recall)
	bodyPath := builder.CreateString(o.BodyPath)

	nFilePaths := len(o.FilePaths)
	var filePaths flatbuffers.UOffsetT
	if nFilePaths != 0 {
		offsets := make([]flatbuffers.UOffsetT, nFilePaths)
		for i, fp := range o.FilePaths {
			offsets[i] = builder.CreateString(fp)
		}
		cronfb.DatasetOptionsStartFilePathsVector(builder, nFilePaths)
		for i := nFilePaths - 1; i >= 0; i-- {
			builder.PrependUOffsetT(offsets[i])
		}
		filePaths = builder.EndVector(nFilePaths)
	}

	// TODO (b5) - encode config & secrets

	cronfb.DatasetOptionsStart(builder)
	cronfb.DatasetOptionsAddCommitTitle(builder, commitTitle)
	cronfb.DatasetOptionsAddCommitMessage(builder, commitMessage)
	cronfb.DatasetOptionsAddRecall(builder, recall)
	cronfb.DatasetOptionsAddBodyPath(builder, bodyPath)
	if nFilePaths != 0 {
		cronfb.DatasetOptionsAddFilePaths(builder, filePaths)
	}

	cronfb.DatasetOptionsAddPublish(builder, o.Publish)
	cronfb.DatasetOptionsAddStrict(builder, o.Strict)
	cronfb.DatasetOptionsAddForce(builder, o.Force)
	cronfb.DatasetOptionsAddConvertFormatToPrev(builder, o.ConvertFormatToPrev)
	cronfb.DatasetOptionsAddNoRender(builder, o.NoRender)

	return cronfb.DatasetOptionsEnd(builder)
}
