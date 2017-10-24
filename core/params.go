package core

type GetParams struct {
	Username string
	Name     string
	Hash     string
}

type ListParams struct {
	OrderBy string
	Limit   int
	Offset  int
}
