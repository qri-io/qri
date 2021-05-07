// +build arm 386

package sql

import (
	"context"
	"errors"
	"io"

	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/repo"
)

// Service represents SQL running
type Service struct{}

// New returns a new Service
func New(r repo.Repo, loader dsref.Loader) *Service {
	return &Service{}
}

// Exec fails to execute on 32-bit systems
func (svc *Service) Exec(ctx context.Context, w io.Writer, outFormat, query string) error {
	return errors.New("sql command is not available on 32-bit systems")
}
