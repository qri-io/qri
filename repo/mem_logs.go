package repo

import (
	"sort"
)

type MemQueryLog []*QueryLogItem

func (ql *MemQueryLog) LogQuery(item *QueryLogItem) error {
	logs := append(*ql, item)
	sort.Slice(logs, func(i, j int) bool { return logs[i].Time.Before(logs[j].Time) })
	*ql = logs
	return nil
}

func (ql *MemQueryLog) QueryLogItem(q *QueryLogItem) (*QueryLogItem, error) {
	for _, item := range *ql {
		if item.DatasetPath.Equal(q.DatasetPath) ||
			item.Query == q.Query ||
			item.Time.Equal(q.Time) ||
			item.Key.Equal(q.Key) {
			return item, nil
		}
	}
	return nil, ErrNotFound
}

func (ql MemQueryLog) ListQueryLogs(limit, offset int) ([]*QueryLogItem, error) {
	if offset > len(ql) {
		offset = len(ql)
	}
	stop := limit + offset
	if stop > len(ql) {
		stop = len(ql)
	}

	return ql[offset:stop], nil
}
