package repo

type MemQueryLog []*DatasetRef

func (ql *MemQueryLog) LogQuery(ref *DatasetRef) error {
	*ql = append(*ql, &DatasetRef{Name: ref.Name, Path: ref.Path})
	return nil
}

func (ql MemQueryLog) GetQueryLogs(limit, offset int) ([]*DatasetRef, error) {
	if offset > len(ql) {
		offset = len(ql)
	}
	stop := limit + offset
	if stop > len(ql) {
		stop = len(ql)
	}

	return ql[offset:stop], nil
}
