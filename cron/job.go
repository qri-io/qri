package cron

import (
	"time"

	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/qri-io/iso8601"
	cron "github.com/qri-io/qri/cron/cron_fbs"
)

// Job represents a "cron job" that can be scheduled for repeated execution at
// a specified Periodicity (time interval)
type Job struct {
	Name        string
	Type        JobType
	LastRun     time.Time
	LastError   string
	Periodicity iso8601.RepeatingInterval
	Secrets     map[string]string
}

// NextExec returns the next time execution horizion. If job periodicity is
// improperly configured, the returned time will be zero
func (job *Job) NextExec() time.Time {
	return job.Periodicity.After(job.LastRun)
}

// MarshalFb writes a job to a builder
func (job *Job) MarshalFb(builder *flatbuffers.Builder) flatbuffers.UOffsetT {
	name := builder.CreateString(job.Name)
	typ := builder.CreateString(string(job.Type))
	lastRun := builder.CreateString(job.LastRun.Format(time.RFC3339))
	lastError := builder.CreateString(job.LastError)
	p := builder.CreateString(job.Periodicity.String())

	cron.JobStart(builder)
	cron.JobAddName(builder, name)
	cron.JobAddType(builder, typ)
	cron.JobAddLastRun(builder, lastRun)
	cron.JobAddLastError(builder, lastError)
	cron.JobAddPeriodicity(builder, p)
	return cron.JobEnd(builder)
}

// UnmarshalFb decodes a job from a flatbuffer
func (job *Job) UnmarshalFb(j *cron.Job) error {
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
		LastRun:     lastRun,
		Type:        JobType(j.Type()),
		LastError:   string(j.LastError()),
		Periodicity: p,
		// TODO (b5) - secrets storages
	}
	return nil
}
