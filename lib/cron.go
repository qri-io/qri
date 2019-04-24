package lib

import (
	"fmt"

	"github.com/qri-io/qri/cron"
)

// NewCronMethods creates a cron handle from an instance
func NewCronMethods(inst *Instance) *CronMethods {
	return &CronMethods{inst: inst}
}

// CronMethods encapsulates business logic for the qri cron service
type CronMethods struct {
	inst *Instance
	cron *cron.Cron
}

func (m *CronMethods) Add() error {
	return fmt.Errorf("not finished")
}

func (m *CronMethods) Remove() error {
	return fmt.Errorf("not finished")
}

func (m *CronMethods) List(p *ListParams, jobs *[]*cron.Job) error {
	return fmt.Errorf("not finished")
}

func (m *CronMethods) Log() error {
	return fmt.Errorf("not finished")
}

func (m *CronMethods) Start() error {
	return fmt.Errorf("not finished")
}

func (m *CronMethods) Stop() error {
	return fmt.Errorf("not finished")
}
