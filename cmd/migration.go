package cmd

type Migration struct {
	Id          string   `json:"id" sql:"id"`
	Created     int64    `json:"created" sql:"created"`
	Updated     int64    `json:"updated" sql:"updated"`
	Dataset     *Dataset `json:"dataset,omitempty" sql:"-"`
	Owner       *User    `json:"owner,omitempty" sql:"-"`
	Number      int64    `json:"number" sql:"number"`
	Description string   `json:"description" sql:"description"`
	Exectued    int64    `json:"exectued,omitempty" sql:"exectued"`
	Declined    int64    `json:"declined,omitempty" sql:"declined"`
	Sql         string   `json:"sql,omitempty" sql:"declined"`
}
