package repo

import (
	"sort"
)

// MemQueryLog is an in-memory implementation of the
// QueryLog interface
type MemQueryLog []*QueryLogItem

// LogQuery adds a query entry to the store
func (ql *MemQueryLog) LogQuery(item *QueryLogItem) error {
	logs := append(*ql, item)
	sort.Slice(logs, func(i, j int) bool { return logs[i].Time.Before(logs[j].Time) })
	*ql = logs
	return nil
}

// QueryLogItem fills a partial QueryLogItem with all details from the store
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

// ListQueryLogs grabs a set of QueryLogItems from the store
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
