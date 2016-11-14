package cmd

import (
	"fmt"
	"os"

	"github.com/qri-io/repo"
	"github.com/spf13/cobra"
)

// ErrExit writes an error to stdout & exits
func ErrExit(err error) {
	fmt.Println(err.Error())
	os.Exit(1)
}

func GetRepo(cmd *cobra.Command, args []string) *repo.Repo {
	return repo.NewRepo(func(o *repo.RepoOpt) {
		o.BasePath = GetWd()
	})
}
