package cmd

import (
	"github.com/qri-io/csvf"
	"github.com/qri-io/ql"
)

// Change is a modification to data on a table in a dataset. To modify a dataset's schema, see migrations.
// Modifications can be inserts, updates, or deletes.
// Migrations & Changes in combination compose the entire history of a dataset.
// Users can prepare change requests, modify them over time, dicuss the change request via comments.
// Ultimately administrators of the dataset either accept or decline the change.
type Change struct {
	Id      string `json:"id" sql:"id"`
	Created int64  `json:"created" sql:"created"`
	Updated int64  `json:"updated" sql:"updated"`
	// the target dataset. required.
	Dataset *Dataset `json:"dataset,omitempty" sql:"dataset_id"`
	// user that created this change. required.
	Owner *User `json:"owner,omitempty" sql:"owner_id"`
	// index of change change request in relation to the target dataset
	Number int64 `json:"number" sql:"number"`
	// comment outlining what the CR does
	Description string `json:"description" sql:"description"`
	// unix timestamp for execution date. 0 = not executed
	Executed int64 `json:"executed,omitempty" sql:"executed"`
	// unix timestamp for declined date. 0 = not declined executed
	Declined int64 `json:"declined,omitempty" sql:"declined"`
	// insertions
	RowsAffected int64 `json:"rows_affected" sql:"rows_affected"`
	// the table this change will affect. required.
	TableName string `json:"table_name" sql:"table_name"`
	// the type of change
	Type ChangeType `json:"type" sql:"type"`
	// an sql statement to execute. either sql or file is required.
	Sql ql.Statement `json:"sql,omitempty" sql:"sql"`
	// a csv file that contains changes. either sql or file is required.
	File *csvf.File `json:"data,omitempty" sql:"-"`
}
