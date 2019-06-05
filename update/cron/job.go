package cron

import (
	"fmt"
	"path/filepath"
	"time"

	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/qri-io/iso8601"
	cronfb "github.com/qri-io/qri/update/cron/cron_fbs"
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
//
// a Job struct has one of three "run" states, which describe it's position in
// the execution lifecycle:
// * unexected: job.RunStart.IsZero() && job.RunStop.IsZero()
// * executing: !job.RunStart.IsZero() && job.RunStop.IsZero()
// * completed: !job.RunStart.IsZero() && !job.RunStop.IsZero()
type Job struct {
	Name         string                    `json:"name"`
	Alias        string                    `json:"alias"`
	Type         JobType                   `json:"type"`
	Periodicity  iso8601.RepeatingInterval `json:"periodicity"`
	PrevRunStart time.Time                 `json:"lastRunStart,omitempty"`

	RunNumber   int64     `json:"runNumber,omitempty"`
	RunStart    time.Time `json:"runStart,omitempty"`
	RunStop     time.Time `json:"runStop,omitempty"`
	RunError    string    `json:"runError,omitempty"`
	LogFilePath string    `json:"logFilePath,omitempty"`

	RepoPath string `json:"repoPath,omitempty"`

	Options Options `json:"options,omitempty"`
}

// zero is a "constant" representing an empty repeating interval
// TODO (b5) - add a IsZero methods to iso8601 structs
var zero iso8601.RepeatingInterval

// Validate confirms a Job contains valid details for scheduling
func (job *Job) Validate() error {
	if job.Name == "" {
		return fmt.Errorf("name is required")
	}

	if job.Periodicity == zero {
		return fmt.Errorf("period is required")
	}
	if job.Type != JTDataset && job.Type != JTShellScript {
		return fmt.Errorf("invalid job type: %s", job.Type)
	}
	return nil
}

// NextExec returns the next time execution horizon. If job periodicity is
// improperly configured, the returned time will be zero
func (job *Job) NextExec() time.Time {
	return job.Periodicity.After(job.PrevRunStart)
}

// LogName returns a canonical name string for a job that's executed and saved
// to a logging system
func (job *Job) LogName() string {
	return fmt.Sprintf("%d-%s", job.RunNumber, filepath.Base(job.Name))
}

// Copy creates a copy of a job
func (job *Job) Copy() *Job {
	cp := &Job{
		Name:         job.Name,
		Alias:        job.Alias,
		Type:         job.Type,
		Periodicity:  job.Periodicity,
		PrevRunStart: job.PrevRunStart,

		RunNumber:   job.RunNumber,
		RunStart:    job.RunStart,
		RunStop:     job.RunStop,
		RunError:    job.RunError,
		LogFilePath: job.LogFilePath,
		RepoPath:    job.RepoPath,
	}

	if job.Options != nil {
		cp.Options = job.Options
	}

	return cp
}

// FlatbufferBytes formats a job as a flatbuffer byte slice
func (job *Job) FlatbufferBytes() []byte {
	builder := flatbuffers.NewBuilder(0)
	off := job.MarshalFlatbuffer(builder)
	builder.Finish(off)
	return builder.FinishedBytes()
}

// MarshalFlatbuffer writes a job to a builder
func (job *Job) MarshalFlatbuffer(builder *flatbuffers.Builder) flatbuffers.UOffsetT {
	name := builder.CreateString(job.Name)
	alias := builder.CreateString(job.Alias)

	prevRunStart := builder.CreateString(job.PrevRunStart.Format(time.RFC3339))
	runStart := builder.CreateString(job.RunStart.Format(time.RFC3339))
	runStop := builder.CreateString(job.RunStop.Format(time.RFC3339))
	lastError := builder.CreateString(job.RunError)
	logPath := builder.CreateString(job.LogFilePath)
	repoPath := builder.CreateString(job.RepoPath)
	p := builder.CreateString(job.Periodicity.String())

	var opts flatbuffers.UOffsetT
	if job.Options != nil {
		opts = job.Options.MarshalFlatbuffer(builder)
	}

	cronfb.JobStart(builder)
	cronfb.JobAddName(builder, name)
	cronfb.JobAddAlias(builder, alias)

	cronfb.JobAddType(builder, job.Type.Enum())
	cronfb.JobAddPeriodicity(builder, p)
	cronfb.JobAddPrevRunStart(builder, prevRunStart)

	cronfb.JobAddRunNumber(builder, job.RunNumber)
	cronfb.JobAddRunStart(builder, runStart)
	cronfb.JobAddRunStop(builder, runStop)
	cronfb.JobAddRunError(builder, lastError)
	cronfb.JobAddLogFilePath(builder, logPath)
	cronfb.JobAddRepoPath(builder, repoPath)
	cronfb.JobAddOptionsType(builder, job.fbOptionsType())
	if opts != 0 {
		cronfb.JobAddOptions(builder, opts)
	}
	return cronfb.JobEnd(builder)
}

func (job *Job) fbOptionsType() byte {
	switch job.Options.(type) {
	case *DatasetOptions:
		return cronfb.OptionsDatasetOptions
	default:
		return 0 // will fire for nil case
	}
}

// UnmarshalFlatbuffer decodes a job from a flatbuffer
func (job *Job) UnmarshalFlatbuffer(j *cronfb.Job) error {
	prevRunStart, err := time.Parse(time.RFC3339, string(j.PrevRunStart()))
	if err != nil {
		return err
	}
	runStart, err := time.Parse(time.RFC3339, string(j.RunStart()))
	if err != nil {
		return err
	}
	runStop, err := time.Parse(time.RFC3339, string(j.RunStop()))
	if err != nil {
		return err
	}

	p, err := iso8601.ParseRepeatingInterval(string(j.Periodicity()))
	if err != nil {
		return err
	}

	*job = Job{
		Name:         string(j.Name()),
		Alias:        string(j.Alias()),
		Type:         JobType(cronfb.EnumNamesJobType[j.Type()]),
		Periodicity:  p,
		PrevRunStart: prevRunStart,

		RunNumber:   j.RunNumber(),
		RunStart:    runStart,
		RunStop:     runStop,
		RunError:    string(j.RunError()),
		LogFilePath: string(j.LogFilePath()),
		RepoPath:    string(j.RepoPath()),
	}

	unionTable := new(flatbuffers.Table)
	if j.Options(unionTable) {
		if j.OptionsType() == cronfb.OptionsDatasetOptions {
			fbOpts := &cronfb.DatasetOptions{}
			fbOpts.Init(unionTable.Bytes, unionTable.Pos)
			opts := &DatasetOptions{}
			opts.UnmarshalFlatbuffer(fbOpts)
			job.Options = opts
		}
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
// TODO (b5) - we should contribute flexbuffer support for golang & remove this entirely
type DatasetOptions struct {
	Title     string
	Message   string
	Recall    string
	BodyPath  string
	FilePaths []string

	Publish             bool
	Strict              bool
	Force               bool
	ConvertFormatToPrev bool
	ShouldRender        bool

	Config  map[string]string
	Secrets map[string]string
}

// MarshalFlatbuffer writes to a builder
func (o *DatasetOptions) MarshalFlatbuffer(builder *flatbuffers.Builder) flatbuffers.UOffsetT {
	commitTitle := builder.CreateString(o.Title)
	commitMessage := builder.CreateString(o.Message)
	recall := builder.CreateString(o.Recall)
	bodyPath := builder.CreateString(o.BodyPath)

	var filePaths flatbuffers.UOffsetT
	nFilePaths := len(o.FilePaths)
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

	var config flatbuffers.UOffsetT
	nConfigs := len(o.Config)
	if nConfigs > 0 {
		offsets := make([]flatbuffers.UOffsetT, nConfigs)
		i := 0
		for key, val := range o.Config {
			keyOff := builder.CreateString(key)
			valOff := builder.CreateString(val)

			cronfb.StringMapValStart(builder)
			cronfb.StringMapValAddKey(builder, keyOff)
			cronfb.StringMapValAddVal(builder, valOff)
			offsets[i] = cronfb.StringMapValEnd(builder)
			i++
		}

		cronfb.DatasetOptionsStartConfigVector(builder, nConfigs)
		for i := nConfigs - 1; i >= 0; i-- {
			builder.PrependUOffsetT(offsets[i])
		}
		config = builder.EndVector(nConfigs)
	}

	var secrets flatbuffers.UOffsetT
	nSecrets := len(o.Secrets)
	if nSecrets > 0 {
		offsets := make([]flatbuffers.UOffsetT, nSecrets)
		i := 0
		for key, val := range o.Secrets {
			keyOff := builder.CreateString(key)
			valOff := builder.CreateString(val)

			cronfb.StringMapValStart(builder)
			cronfb.StringMapValAddKey(builder, keyOff)
			cronfb.StringMapValAddVal(builder, valOff)
			offsets[i] = cronfb.StringMapValEnd(builder)
			i++
		}

		cronfb.DatasetOptionsStartSecretsVector(builder, len(offsets))
		for i := nSecrets - 1; i >= 0; i-- {
			builder.PrependUOffsetT(offsets[i])
		}
		secrets = builder.EndVector(nSecrets)
	}

	cronfb.DatasetOptionsStart(builder)
	cronfb.DatasetOptionsAddTitle(builder, commitTitle)
	cronfb.DatasetOptionsAddMessage(builder, commitMessage)
	cronfb.DatasetOptionsAddRecall(builder, recall)
	cronfb.DatasetOptionsAddBodyPath(builder, bodyPath)

	cronfb.DatasetOptionsAddPublish(builder, o.Publish)
	cronfb.DatasetOptionsAddStrict(builder, o.Strict)
	cronfb.DatasetOptionsAddForce(builder, o.Force)
	cronfb.DatasetOptionsAddConvertFormatToPrev(builder, o.ConvertFormatToPrev)
	cronfb.DatasetOptionsAddShouldRender(builder, o.ShouldRender)

	cronfb.DatasetOptionsAddFilePaths(builder, filePaths)
	cronfb.DatasetOptionsAddConfig(builder, config)
	cronfb.DatasetOptionsAddSecrets(builder, secrets)

	return cronfb.DatasetOptionsEnd(builder)
}

// UnmarshalFlatbuffer reads flatbuffer data into DatasetOptions
func (o *DatasetOptions) UnmarshalFlatbuffer(fbo *cronfb.DatasetOptions) {
	o.Title = string(fbo.Title())
	o.Message = string(fbo.Message())
	o.Recall = string(fbo.Recall())
	o.BodyPath = string(fbo.BodyPath())

	if fbo.FilePathsLength() > 0 {
		o.FilePaths = make([]string, fbo.FilePathsLength())
		for i := range o.FilePaths {
			o.FilePaths[i] = string(fbo.FilePaths(i))
		}
	}

	o.Publish = fbo.Publish()
	o.Strict = fbo.Strict()
	o.Force = fbo.Force()
	o.ConvertFormatToPrev = fbo.ConvertFormatToPrev()
	o.ShouldRender = fbo.ShouldRender()

	// TODO (b5): unmarshal secrets & config:
	// Config  map[string]string
	// Secrets map[string]string
	if fbo.ConfigLength() > 0 {
		o.Config = map[string]string{}
		var val cronfb.StringMapVal
		for i := 0; i < fbo.ConfigLength(); i++ {
			if fbo.Config(&val, i) {
				o.Config[string(val.Key())] = string(val.Val())
			}
		}
	}

	if fbo.SecretsLength() > 0 {
		o.Secrets = map[string]string{}
		var val cronfb.StringMapVal
		for i := 0; i < fbo.SecretsLength(); i++ {
			if fbo.Secrets(&val, i) {
				o.Secrets[string(val.Key())] = string(val.Val())
			}
		}
	}
}
