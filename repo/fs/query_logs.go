package fs_repo

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sort"

	"github.com/qri-io/cafs"
	"github.com/qri-io/qri/repo"
)

type QueryLog struct {
	basepath
	file  File
	store cafs.Filestore
}

func NewQueryLog(base string, file File, store cafs.Filestore) QueryLog {
	return QueryLog{basepath: basepath(base), file: file, store: store}
}

func (ql QueryLog) LogQuery(item *repo.QueryLogItem) error {
	log, err := ql.logs()
	if err != nil {
		return err
	}
	log = append(log, item)
	sort.Slice(log, func(i, j int) bool { return log[i].Time.Before(log[j].Time) })
	return ql.saveFile(log, ql.file)
}

func (ql QueryLog) QueryLogItem(q *repo.QueryLogItem) (*repo.QueryLogItem, error) {
	log, err := ql.logs()
	if err != nil {
		return nil, err
	}

	for _, item := range log {
		if item.DatasetPath.Equal(q.DatasetPath) ||
			item.Query == q.Query ||
			item.Time.Equal(q.Time) ||
			item.Key.Equal(q.Key) {
			return item, nil
		}
	}
	return nil, repo.ErrNotFound
}

func (ql QueryLog) ListQueryLogs(limit, offset int) ([]*repo.QueryLogItem, error) {
	logs, err := ql.logs()
	if err != nil {
		return nil, err
	}

	if offset > len(logs) {
		offset = len(logs)
	}
	stop := limit + offset
	if stop > len(logs) {
		stop = len(logs)
	}

	return logs[offset:stop], nil
}

func (r *QueryLog) logs() ([]*repo.QueryLogItem, error) {
	ds := []*repo.QueryLogItem{}
	data, err := ioutil.ReadFile(r.filepath(r.file))
	if err != nil {
		if os.IsNotExist(err) {
			return ds, nil
		}
		return ds, fmt.Errorf("error loading logs: %s", err.Error())
	}

	if err := json.Unmarshal(data, &ds); err != nil {
		return ds, fmt.Errorf("error unmarshaling logs: %s", err.Error())
	}
	return ds, nil
}
