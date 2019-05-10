// Package update defines the update service
package update

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"

	golog "github.com/ipfs/go-log"
	"github.com/qri-io/dataset"
	"github.com/qri-io/ioes"
	"github.com/qri-io/iso8601"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/cron"
)

var log = golog.Logger("update")

// Start starts the update service
func Start(ctx context.Context, repoPath string, updateCfg *config.Update, daemonize bool) error {
	if updateCfg == nil {
		updateCfg = config.DefaultUpdate()
	}

	cli := cron.HTTPClient{Addr: updateCfg.Address}
	if err := cli.Ping(); err == nil {
		return fmt.Errorf("service already running")
	}

	if daemonize {
		return daemonInstall()
	}

	return start(ctx, repoPath, updateCfg)
}

// StopDaemon checks for a running daemon, uninstalling it if one exists
func StopDaemon() error {
	return daemonUninstall()
}

// Status returns the status details of the update service
func Status() (string, error) {
	return daemonShow()
}

func start(ctx context.Context, repoPath string, updateCfg *config.Update) error {
	var jobStore, logStore cron.JobStore
	switch updateCfg.Type {
	case "fs":
		jobStore = cron.NewFlatbufferJobStore(repoPath + "/cron_jobs.qfb")
		logStore = cron.NewFlatbufferJobStore(repoPath + "/cron_logs.qfb")
	case "mem":
		jobStore = &cron.MemJobStore{}
		logStore = &cron.MemJobStore{}
	default:
		return fmt.Errorf("unknown cron type: %s", updateCfg.Type)
	}

	svc := cron.NewCron(jobStore, logStore, Factory)
	log.Debug("starting update service")
	go func() {
		if err := svc.ServeHTTP(updateCfg.Address); err != nil {
			log.Errorf("starting cron http server: %s", err)
		}
	}()

	return svc.Start(ctx)
}

// Factory returns a function that can run jobs
func Factory(context.Context) cron.RunJobFunc {
	return func(ctx context.Context, streams ioes.IOStreams, job *cron.Job) error {
		log.Debugf("running update: %s", job.Name)

		cmd := JobToCmd(streams, job)
		if cmd == nil {
			return fmt.Errorf("unrecognized update type: %s", job.Type)
		}
		return cmd.Run()
	}
}

// JobToCmd returns an operating system command that will execute the given job
// wiring operating system in/out/errout to the provided iostreams.
func JobToCmd(streams ioes.IOStreams, job *cron.Job) *exec.Cmd {
	switch job.Type {
	case cron.JTDataset:
		return datasetSaveCmd(streams, job)
	case cron.JTShellScript:
		return shellScriptCmd(streams, job)
	default:
		return nil
	}
}

// datasetSaveCmd configures a "qri save" command based on job details
// wiring operating system in/out/errout to the provided iostreams.
func datasetSaveCmd(streams ioes.IOStreams, job *cron.Job) *exec.Cmd {
	args := []string{"save", job.Name}

	if o, ok := job.Options.(*cron.DatasetOptions); ok {
		if o.Title != "" {
			args = append(args, fmt.Sprintf(`--title=%s`, o.Title))
		}
		if o.Message != "" {
			args = append(args, fmt.Sprintf(`--message=%s`, o.Message))
		}
		if o.Recall != "" {
			args = append(args, fmt.Sprintf(`--recall=%s`, o.Recall))
		}
		if o.BodyPath != "" {
			args = append(args, fmt.Sprintf(`--body=%s`, o.BodyPath))
		}
		if len(o.FilePaths) > 0 {
			for _, path := range o.FilePaths {
				args = append(args, fmt.Sprintf(`--file=%s`, path))
			}
		}

		// TODO (b5) - config and secrets

		boolFlags := map[string]bool{
			"--publish":     o.Publish,
			"--strict":      o.Strict,
			"--force":       o.Force,
			"--keep-format": o.ConvertFormatToPrev,
			"--no-render":   !o.ShouldRender,
		}
		for flag, use := range boolFlags {
			if use {
				args = append(args, flag)
			}
		}

	}

	cmd := exec.Command("qri", args...)
	cmd.Stderr = streams.ErrOut
	cmd.Stdout = streams.Out
	cmd.Stdin = streams.In
	return cmd
}

// shellScriptCmd creates an exec.Cmd, wires operating system in/out/errout
// to the provided iostreams.
// Commands are executed with access to the same enviornment variables as the
// process the runner is executing in
func shellScriptCmd(streams ioes.IOStreams, job *cron.Job) *exec.Cmd {
	// TODO (b5) - config and secrets as env vars

	cmd := exec.Command(job.Name)
	cmd.Stderr = streams.ErrOut
	cmd.Stdout = streams.Out
	cmd.Stdin = streams.In
	return cmd
}

// PossibleShellScript checks a path to see if it might be a shell script
// TODO (b5) - deal with platforms that don't use '.sh' as a script extension (windows?)
func PossibleShellScript(path string) bool {
	return filepath.Ext(path) == ".sh"
}

// DatasetToJob converts a dataset to cron.Job
func DatasetToJob(ds *dataset.Dataset, periodicity string, opts *cron.DatasetOptions) (job *cron.Job, err error) {
	if periodicity == "" && ds.Meta != nil && ds.Meta.AccrualPeriodicity != "" {
		periodicity = ds.Meta.AccrualPeriodicity
	}

	if periodicity == "" {
		return nil, fmt.Errorf("scheduling dataset updates requires a meta component with accrualPeriodicity set")
	}

	p, err := iso8601.ParseRepeatingInterval(periodicity)
	if err != nil {
		return nil, err
	}

	job = &cron.Job{
		// TODO (b5) - dataset.Dataset needs an Alias() method:
		Name:         fmt.Sprintf("%s/%s", ds.Peername, ds.Name),
		Periodicity:  p,
		Type:         cron.JTDataset,
		LastRunStart: ds.Commit.Timestamp,
		LastRunStop:  ds.Commit.Timestamp,
	}
	if opts != nil {
		job.Options = opts
	}
	err = job.Validate()

	return
}

// ShellScriptToJob turns a shell script into cron.Job
func ShellScriptToJob(f qfs.File, periodicity string, opts *cron.ShellScriptOptions) (job *cron.Job, err error) {
	p, err := iso8601.ParseRepeatingInterval(periodicity)
	if err != nil {
		return nil, err
	}

	job = &cron.Job{
		Name:        f.FullPath(),
		Periodicity: p,
		Type:        cron.JTShellScript,
	}
	if opts != nil {
		job.Options = opts
	}
	err = job.Validate()
	return
}
