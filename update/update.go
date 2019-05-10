// Package update defines the update service
package update

import (
	"context"
	"fmt"

	golog "github.com/ipfs/go-log"
	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/base"
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
		return DaemonInstall()
	}

	return start(ctx, repoPath, updateCfg)

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
		cmd := base.JobToCmd(streams, job)
		if cmd == nil {
			return fmt.Errorf("unrecognized update type: %s", job.Type)
		}
		return cmd.Run()
	}
}
