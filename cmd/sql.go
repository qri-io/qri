package cmd

import (
	"bytes"

	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/lib"
	"github.com/spf13/cobra"
)

// NewSQLCommand creates a new `qri sql` command for running SQL on datasets
func NewSQLCommand(f Factory, ioStreams ioes.IOStreams) *cobra.Command {
	o := &SQLOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:   "sql QUERY",
		Short: "run a SQL query on local dataset(s)",
		Long: `sql runs Structured Query Language (SQL) commands, using local datasets 
as tables.

The qri sql command differs from classic relational databases like MySQL or
PostgreSQL in a few ways:
  * sql queries datasets as if they were tables, Any valid dataset reference
    can be used as a table name
  * to query a dataset, it must be in your local qri repo
  * Tables must always be aliased. eg: select a.col from user/dataset as a
  * For a dataset to be queryable it's schema must be properly configured to
    describe a tabular structure, with valid column names & types
  * Referencing columns that do not exist will return null values instead of
    throwing an error`,
		Example: `  # first, fetch the dataset b5/world_bank_population:
  $ qri add b5/world_bank_population
  $ qri sql "SELECT 
    wbp.country_name, wbp.year_2018
    FROM b5/world_bank_population as wbp"

  # join b5/world_bank_population with b5/country_codes
  $ qri add b5/country_codes
  $ qri sql "
    SELECT 
    cc.official_name_en, wbp.year_2010, wbp.year_2011 
    FROM b5/world_bank_population as wbp
    LEFT JOIN b5/country_codes as cc 
    ON cc.iso_3166_1_alpha_3 = wbp.country_code"`,
		Annotations: map[string]string{
			"group": "dataset",
		},
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.Complete(f, args); err != nil {
				return err
			}
			return o.Run()
		},
	}

	cmd.Flags().StringVarP(&o.Format, "format", "f", "table", "set output format [table]")

	return cmd
}

// SQLOptions encapsulates state for the SQL command
type SQLOptions struct {
	ioes.IOStreams

	Query  string
	Format string

	SQLMethods *lib.SQLMethods
}

// Complete adds any missing configuration that can only be added just before
// calling Run
func (o *SQLOptions) Complete(f Factory, args []string) (err error) {
	o.Query = args[0]
	o.SQLMethods, err = f.SQLMethods()
	return
}

// Run executes the search command
func (o *SQLOptions) Run() (err error) {
	o.StartSpinner()

	p := &lib.SQLQueryParams{
		Query:        o.Query,
		OutputFormat: o.Format,
	}

	res := []byte{}
	if err := o.SQLMethods.Exec(p, &res); err != nil {
		o.StopSpinner()
		return err
	}

	o.StopSpinner()
	printToPager(o.Out, bytes.NewBuffer(res))
	return nil
}
