package list

type Params struct {
	Filter  []string
	OrderBy []string
	Limit   int
	Offset  int
}

func (lp Params) WithFilter(f ...string) Params {
	return Params{
		Filter:  f,
		OrderBy: lp.OrderBy,
		Offset:  lp.Offset,
		Limit:   lp.Limit,
	}
}

func (lp Params) WithOffsetLimit(offset, limit int) Params {
	return Params{
		Filter:  lp.Filter,
		OrderBy: lp.OrderBy,
		Offset:  offset,
		Limit:   limit,
	}
}

func (lp Params) WithOrderBy(ob ...string) Params {
	return Params{
		Filter:  lp.Filter,
		OrderBy: ob,
		Offset:  lp.Offset,
		Limit:   lp.Limit,
	}
}
