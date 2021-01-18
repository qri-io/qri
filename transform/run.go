package transform

import (
	"time"

	"github.com/google/uuid"
)

// NewRunID creates a run identifier
func NewRunID() string {
	return uuid.New().String()
}

// Run represents the state of a transform execution
type Run struct {
	ID          string     `json:"ID"`
	Start       *time.Time `json:"start"`
	Stop        *time.Time `json:"stop"`
	Error       string     `json:"error,omitempty"`
	LogFilePath string     `json:"logFilePath,omitempty"`
}

// NewRun constructs a run pointer
func NewRun() *Run {
	return &Run{
		ID: NewRunID(),
	}
}
