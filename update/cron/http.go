package cron

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	apiutil "github.com/qri-io/apiutil"
	flatbuffers "github.com/google/flatbuffers/go"
	cronfb "github.com/qri-io/qri/update/cron/cron_fbs"
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
// established at all, Ping will return ErrUnreachable
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
	return maybeErrorResponse(res)
}

// ListJobs  jobs by querying an HTTP server
func (c HTTPClient) ListJobs(ctx context.Context, offset, limit int) ([]*Job, error) {
	res, err := http.Get(fmt.Sprintf("http://%s/jobs?offset=%d&limit=%d", c.Addr, offset, limit))
	if err != nil {
		return nil, err
	}

	return decodeListJobsResponse(res)
}

// Job gets a job by querying an HTTP server
func (c HTTPClient) Job(ctx context.Context, name string) (*Job, error) {
	res, err := http.Get(fmt.Sprintf("http://%s/job?name=%s", c.Addr, name))
	if err != nil {
		return nil, err
	}

	if res.StatusCode == 200 {
		return decodeJobResponse(res)
	}

	return nil, maybeErrorResponse(res)
}

// Schedule adds a job to the cron scheduler via an HTTP request
func (c HTTPClient) Schedule(ctx context.Context, job *Job) error {
	return c.postJob(job)
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

	return maybeErrorResponse(res)
}

// ListLogs gives a log of executed jobs
func (c HTTPClient) ListLogs(ctx context.Context, offset, limit int) ([]*Job, error) {
	res, err := http.Get(fmt.Sprintf("http://%s/logs?offset=%d&limit=%d", c.Addr, offset, limit))
	if err != nil {
		return nil, err
	}

	return decodeListJobsResponse(res)
}

// Log returns a single executed job by job.LogName
func (c HTTPClient) Log(ctx context.Context, logName string) (*Job, error) {
	res, err := http.Get(fmt.Sprintf("http://%s/log?log_name=%s", c.Addr, logName))
	if err != nil {
		return nil, err
	}

	if res.StatusCode == 200 {
		return decodeJobResponse(res)
	}

	return nil, maybeErrorResponse(res)
}

// LogFile returns a reader for a file at the given name
func (c HTTPClient) LogFile(ctx context.Context, logName string) (io.ReadCloser, error) {
	res, err := http.Get(fmt.Sprintf("http://%s/log/output?log_name=%s", c.Addr, logName))
	if err != nil {
		return nil, err
	}

	if res.StatusCode == 200 {
		return res.Body, nil
	}

	return nil, maybeErrorResponse(res)
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

	return maybeErrorResponse(res)
}

func maybeErrorResponse(res *http.Response) error {
	if res.StatusCode == 200 {
		return nil
	}

	errData, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	return fmt.Errorf(string(errData))
}

func decodeListJobsResponse(res *http.Response) ([]*Job, error) {
	defer res.Body.Close()
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	js := cronfb.GetRootAsJobs(data, 0)
	dec := &cronfb.Job{}
	jobs := make([]*Job, js.ListLength())

	for i := 0; i < js.ListLength(); i++ {
		if js.List(dec, i) {
			decJob := &Job{}
			if err := decJob.UnmarshalFlatbuffer(dec); err != nil {
				return nil, err
			}
			jobs[i] = decJob
		}
	}

	return jobs, nil
}

func decodeJobResponse(res *http.Response) (*Job, error) {
	defer res.Body.Close()
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	js := cronfb.GetRootAsJob(data, 0)
	dec := &Job{}
	err = dec.UnmarshalFlatbuffer(js)
	return dec, err
}

// ServeHTTP spins up an HTTP server at the specified address
func (c *Cron) ServeHTTP(addr string) error {
	s := &http.Server{
		Addr:    addr,
		Handler: newCronRoutes(c),
	}
	return s.ListenAndServe()
}

func newCronRoutes(c *Cron) http.Handler {
	m := http.NewServeMux()
	m.HandleFunc("/", c.statusHandler)
	m.HandleFunc("/jobs", c.jobsHandler)
	m.HandleFunc("/job", c.jobHandler)
	m.HandleFunc("/logs", c.logsHandler)
	m.HandleFunc("/log", c.loggedJobHandler)
	m.HandleFunc("/log/output", c.loggedJobFileHandler)
	m.HandleFunc("/run", c.runHandler)

	return m
}

func (c *Cron) statusHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (c *Cron) jobsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		// TODO (b5): handle these errors, but they'll default to 0 so it's mainly
		// for reporting when we're given odd values
		offset, _ := apiutil.ReqParamInt("offset", r)
		limit, _ := apiutil.ReqParamInt("limit", r)

		js, err := c.ListJobs(r.Context(), offset, limit)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Write(jobs(js).FlatbufferBytes())
		return

	case "POST":
		data, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		j := cronfb.GetRootAsJob(data, 0)
		job := &Job{}
		if err := job.UnmarshalFlatbuffer(j); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}

		if err := c.schedule.PutJob(r.Context(), job); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

	case "DELETE":
		name := r.FormValue("name")
		if err := c.Unschedule(r.Context(), name); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
	}

}

func (c *Cron) jobHandler(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	job, err := c.Job(r.Context(), name)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	w.Write(job.FlatbufferBytes())
}

func (c *Cron) logsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {

	case "GET":
		// TODO (b5): handle these errors, but they'll default to 0 so it's mainly
		// for reporting when we're given odd values
		offset, _ := apiutil.ReqParamInt("offset", r)
		limit, _ := apiutil.ReqParamInt("limit", r)

		log, err := c.ListLogs(r.Context(), offset, limit)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Write(jobs(log).FlatbufferBytes())
		return

	}
}

func (c *Cron) loggedJobHandler(w http.ResponseWriter, r *http.Request) {
	logName := r.FormValue("log_name")
	job, err := c.Log(r.Context(), logName)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	w.Write(job.FlatbufferBytes())
}

func (c *Cron) loggedJobFileHandler(w http.ResponseWriter, r *http.Request) {
	logName := r.FormValue("log_name")
	f, err := c.LogFile(r.Context(), logName)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	io.Copy(w, f)
	return
}

func (c *Cron) runHandler(w http.ResponseWriter, r *http.Request) {
	// TODO (b5): implement an HTTP run handler
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte("not finished"))
	// c.runJob(r.Context(), nil)
}
