package cron

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs"
	cronfb "github.com/qri-io/qri/cron/cron_fbs"
)

// HTTPClient implements the Scheduler interface over HTTP, talking to a
// Cron HTTPServer
type HTTPClient struct {
	Addr string
}

// assert HTTPClient is a Scheduler at compile time
var _ Scheduler = (*HTTPClient)(nil)

// ErrUnreachable defines errors where the server cannot be reached
// TODO (b5): consider moving this to qfs
var ErrUnreachable = fmt.Errorf("cannot establish a connection to the server")

// Ping confirms client can dial the server, if a connection cannot be
// established at all, Ping will return ErrUnreachable, all other errors
// will
func (c HTTPClient) Ping() error {
	res, err := http.Get(fmt.Sprintf("http://%s", c.Addr))
	if err != nil {
		msg := strings.ToLower(err.Error())

		// TODO (b5): a number of errors constitute a service being "unreachable",
		// we should make a more exhaustive assessment. common errors already covered:
		// "connect: Connection refused"
		// "dial tcp: lookup [url] no such host"
		if strings.Contains(msg, "refused") || strings.Contains(msg, "no such host") {
			return ErrUnreachable
		}
		return err
	}

	if res.StatusCode == http.StatusOK {
		return nil
	}
	return resError(res)
}

// Jobs lists jobs by querying an HTTP server
func (c HTTPClient) Jobs(ctx context.Context, offset, limit int) ([]*Job, error) {
	res, err := http.Get(fmt.Sprintf("http://%s/jobs?offset=%d&limit=&%d", c.Addr, offset, limit))
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	js := cronfb.GetRootAsJobs(data, 0)
	dec := &cronfb.Job{}
	jobs := make([]*Job, js.ListLength())

	for i := 0; i < js.ListLength(); i++ {
		js.List(dec, i)
		decJob := &Job{}
		if err := decJob.UnmarshalFlatbuffer(dec); err != nil {
			return nil, err
		}
		jobs[i] = decJob
	}

	return jobs, nil
}

// Job gets a job by querying an HTTP server
func (c HTTPClient) Job(ctx context.Context, name string) (*Job, error) {
	return nil, fmt.Errorf("not finished")
}

// ScheduleDataset adds a dataset job by querying an HTTP server
func (c HTTPClient) ScheduleDataset(ctx context.Context, ds *dataset.Dataset, periodicity string, opts *DatasetOptions) (*Job, error) {
	job, err := datasetToJob(ds, periodicity, opts)
	if err != nil {
		return nil, err
	}
	return job, c.postJob(job)
}

// ScheduleShellScript adds a shellscript job by querying an HTTP server
func (c HTTPClient) ScheduleShellScript(ctx context.Context, f qfs.File, periodicity string, opts *ShellScriptOptions) (*Job, error) {
	job, err := shellScriptToJob(f, periodicity, opts)
	if err != nil {
		return nil, err
	}
	return job, c.postJob(job)
}

// Unschedule removes a job from scheduling
func (c HTTPClient) Unschedule(ctx context.Context, name string) error {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("http://%s/jobs?name=%s", c.Addr, name), nil)
	if err != nil {
		return err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	return resError(res)
}

func (c HTTPClient) postJob(job *Job) error {
	builder := flatbuffers.NewBuilder(0)
	off := job.MarshalFlatbuffer(builder)
	builder.Finish(off)
	body := bytes.NewReader(builder.FinishedBytes())

	res, err := http.Post(fmt.Sprintf("http://%s/jobs", c.Addr), "", body)
	if err != nil {
		return err
	}

	return resError(res)
}

func resError(res *http.Response) error {
	if res.StatusCode == 200 {
		return nil
	}

	errData, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	return fmt.Errorf(string(errData))
}
