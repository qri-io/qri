// Package stats calculates statistical metadata for a given dataset
package stats

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strconv"

	logger "github.com/ipfs/go-log"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsio"
)

var (
	// StopFreqCountThreshold is the number of unique values past which we will
	// stop keeping frequencies. This is a simplistic line of defense against
	// unweildly memory consumption
	StopFreqCountThreshold = 10000

	// package logger
	log = logger.Logger("stats")
)

// Stats can generate an array of statistical info for a dataset
type Stats struct {
	cache Cache
}

// New allocates a Stats service
func New(cache Cache) *Stats {
	if cache == nil {
		return &Stats{
			cache: nilCache(false),
		}
	}
	return &Stats{
		cache: cache,
	}
}

// JSON gets stats data as reader of JSON-formatted bytes
func (s *Stats) JSON(ctx context.Context, ds *dataset.Dataset) (r io.Reader, err error) {
	// check cache if there is a Path
	// TODO (ramfox): when we are calculating stats on fsi linked
	// datasets, we need a different metric other the `dataset.Path` to
	// identify and store stats, since FSI datasets will not have
	// a `dataset.Path`. This metric should perhaps come out of the
	// `dataset.BodyFile()` since we must have a bodyFile in order to
	// calculate the stats
	if ds.Path != "" {
		if r, err := s.cache.JSON(ctx, ds.Path); err == nil {
			return r, nil
		}
	}

	body := ds.BodyFile()
	if body == nil {
		return nil, fmt.Errorf("stats: dataset has no body file")
	}
	if ds.Structure == nil {
		return nil, fmt.Errorf("stats: dataset is missing structure")
	}

	rdr, err := dsio.NewEntryReader(ds.Structure, ds.BodyFile())
	if err != nil {
		return nil, err
	}

	acc := NewAccumulator(rdr)
	for {
		if _, err := acc.ReadEntry(); err != nil {
			if err.Error() == "EOF" {
				break
			}
			return nil, err
		}
	}
	acc.Close()

	data, err := json.Marshal(ToMap(acc))
	if err != nil {
		return nil, err
	}

	if ds.Path != "" {
		go func() {
			if err := s.cache.PutJSON(context.Background(), ds.Path, bytes.NewReader(data)); err != nil {
				log.Debugf("putting stats in cache: %v", err.Error())
			}
		}()
	}

	return bytes.NewReader(data), nil
}

// Statser produces a slice of Stat objects
type Statser interface {
	Stats() []Stat
}

// ToMap converts stats to a Plain Old Data object
func ToMap(s Statser) []map[string]interface{} {
	stats := s.Stats()
	if stats == nil {
		return nil
	}

	sm := make([]map[string]interface{}, len(stats))
	for i, stat := range stats {
		sm[i] = stat.Map()
		sm[i]["type"] = stat.Type()
	}
	return sm
}

// Stat describes common features of all statistical types
type Stat interface {
	// Type returns a string identifier for the kind of statistic being reported
	Type() string
	// Map reports statistical details as a map, map must not return nil
	Map() map[string]interface{}
}

// Accumulator wraps a dsio.EntryReader, on each call to read stats
// will update it's internal statistics
// Consumers can only assume the return value of Accumulator.Stats is final
// after a call to Close
type Accumulator struct {
	r     dsio.EntryReader
	stats accumulator
}

var (
	// compile time assertions that Accumulator is an EntryReader & Statser
	_ dsio.EntryReader = (*Accumulator)(nil)
	_ Statser          = (*Accumulator)(nil)
)

// NewAccumulator wraps an entry reader to create a stat accumulator
func NewAccumulator(r dsio.EntryReader) *Accumulator {
	return &Accumulator{r: r}
}

// Stats gets the statistics created by the accumulator
func (r *Accumulator) Stats() []Stat {
	if r.stats == nil {
		return nil
	}
	if stats, ok := r.stats.(Statser); ok {
		return stats.Stats()
	}
	return []Stat{r.stats}
}

// Structure gives the structure being read
func (r *Accumulator) Structure() *dataset.Structure {
	return r.r.Structure()
}

// ReadEntry reads one row of structured data from the reader
func (r *Accumulator) ReadEntry() (dsio.Entry, error) {
	ent, err := r.r.ReadEntry()
	if err != nil {
		return ent, err
	}
	if r.stats == nil {
		r.stats = newAccumulator(ent.Value)
	}
	r.stats.Write(ent)
	return ent, nil
}

// Close finalizes the Reader
func (r *Accumulator) Close() error {
	r.stats.Close()
	return r.r.Close()
}

// accumulator is the common internal inferface for creating a stat
// this package defines at least one accumulator for all values qri works with
// accumulators are one-way state machines that update with each Write
type accumulator interface {
	Stat
	Write(ent dsio.Entry)
	Close()
}

func newAccumulator(val interface{}) accumulator {
	switch val.(type) {
	default:
		return &nullAcc{}
	case float64, float32:
		return newNumericAcc("number")
	case int, int32, int64:
		return newNumericAcc("integer")
	case string:
		return newStringAcc()
	case bool:
		return &boolAcc{}
	case map[string]interface{}:
		return &objectAcc{children: map[string]accumulator{}}
	case []interface{}:
		return &arrayAcc{}
	}
}

type objectAcc struct {
	children map[string]accumulator
}

var (
	_ accumulator = (*objectAcc)(nil)
	_ Statser     = (*objectAcc)(nil)
)

// Stats gets child stats of the accumulator as a Stat slice
func (acc *objectAcc) Stats() (stats []Stat) {
	stats = make([]Stat, len(acc.children))
	keys := make([]string, len(acc.children))
	i := 0
	for key := range acc.children {
		keys[i] = key
		i++
	}
	sort.StringSlice(keys).Sort()
	for j, key := range keys {
		stats[j] = keyedStat{Stat: acc.children[key], key: key}
	}
	return stats
}

// Type indicates this stat accumulator kind
func (acc *objectAcc) Type() string { return "object" }

// Write adds an entry to the stat accumulator
func (acc *objectAcc) Write(e dsio.Entry) {
	if mapEntry, ok := e.Value.(map[string]interface{}); ok {
		for key, val := range mapEntry {
			if _, ok := acc.children[key]; !ok {
				acc.children[key] = newAccumulator(val)
			}
			acc.children[key].Write(dsio.Entry{Key: key, Value: val})
		}
	}
}

// Map formats stat values as a map
func (acc *objectAcc) Map() map[string]interface{} {
	vals := map[string]interface{}{}
	for key, val := range acc.children {
		vals[key] = val.Map()
	}
	return vals
}

// Close finalizes the accumulator
func (acc *objectAcc) Close() {
	for _, val := range acc.children {
		val.Close()
	}
}

type arrayAcc struct {
	children []accumulator
}

var (
	_ accumulator = (*arrayAcc)(nil)
	_ Statser     = (*arrayAcc)(nil)
)

// Stats gets child stats of the array accumulator
func (acc *arrayAcc) Stats() (stats []Stat) {
	stats = make([]Stat, len(acc.children))
	for i, ch := range acc.children {
		stats[i] = ch
	}
	return stats
}

// Type indicates this stat accumulator kind
func (acc *arrayAcc) Type() string { return "array" }

// Write adds an entry to the stat accumulator
func (acc *arrayAcc) Write(e dsio.Entry) {
	if arrayEntry, ok := e.Value.([]interface{}); ok {
		for i, val := range arrayEntry {
			if len(acc.children) == i {
				acc.children = append(acc.children, newAccumulator(val))
			}
			acc.children[i].Write(dsio.Entry{Index: i, Value: val})
		}
	}
}

// Map formats stat values as a map
func (acc *arrayAcc) Map() map[string]interface{} {
	vals := make([]map[string]interface{}, len(acc.children))
	for i, val := range acc.children {
		vals[i] = val.Map()
	}
	// TODO (b5) -  this is silly
	return map[string]interface{}{"values": vals}
}

// Close finalizes the accumulator
func (acc *arrayAcc) Close() {
	for _, val := range acc.children {
		val.Close()
	}
}

const maxUint = ^uint(0)
const maxInt = int(maxUint >> 1)
const minInt = -maxInt - 1

type numericAcc struct {
	typ         string
	count       int
	min         float64
	max         float64
	unique      int
	frequencies map[float64]int
}

var _ accumulator = (*numericAcc)(nil)

func newNumericAcc(typ string) *numericAcc {
	return &numericAcc{
		typ:         typ,
		max:         float64(minInt),
		min:         float64(maxInt),
		frequencies: map[float64]int{},
	}
}

// Type indicates this stat accumulator kind
func (acc *numericAcc) Type() string { return "numeric" }

// Write adds an entry to the stat accumulator
func (acc *numericAcc) Write(e dsio.Entry) {
	var v float64
	switch x := e.Value.(type) {
	case int:
		v = float64(x)
	case int32:
		v = float64(x)
	case int64:
		v = float64(x)
	case float32:
		v = float64(x)
	case float64:
		v = x
	default:
		return
	}

	if acc.frequencies != nil {
		acc.frequencies[v]++
		if len(acc.frequencies) >= StopFreqCountThreshold {
			acc.frequencies = nil
		}
	}

	acc.count++
	if v > acc.max {
		acc.max = v
	}
	if v < acc.min {
		acc.min = v
	}
}

// Map formats stat values as a map
func (acc *numericAcc) Map() map[string]interface{} {
	if acc.count == 0 {
		// avoid reporting default max/min figures, if count is above 0
		// at least one entry has been checked
		return map[string]interface{}{"count": 0}
	}
	m := map[string]interface{}{
		"count": acc.count,
		"min":   acc.min,
		"max":   acc.max,
	}

	if acc.unique != 0 {
		m["unique"] = acc.unique
	}

	if acc.frequencies != nil {
		// need to convert keys to strings b/c many serialization formats aren't
		// down with numeric map keys
		strFrq := map[string]int{}
		for fl, freq := range acc.frequencies {
			strFrq[strconv.FormatFloat(fl, 'f', -1, 64)] = freq
		}
		m["frequencies"] = strFrq
	}

	return m
}

// Close finalizes the accumulator
func (acc *numericAcc) Close() {
	if acc.frequencies != nil {
		// determine unique values
		for key, freq := range acc.frequencies {
			if freq == 1 {
				acc.unique++
				delete(acc.frequencies, key)
			}
		}
		if len(acc.frequencies) == 0 {
			acc.frequencies = nil
		}
	}
}

type stringAcc struct {
	count       int
	minLength   int
	maxLength   int
	unique      int
	frequencies map[string]int
}

var _ accumulator = (*stringAcc)(nil)

func newStringAcc() *stringAcc {
	return &stringAcc{
		maxLength:   minInt,
		minLength:   maxInt,
		frequencies: map[string]int{},
	}
}

// Type indicates this stat accumulator kind
func (acc *stringAcc) Type() string { return "string" }

// Write adds an entry to the stat accumulator
func (acc *stringAcc) Write(e dsio.Entry) {
	if str, ok := e.Value.(string); ok {
		acc.count++

		if acc.frequencies != nil {
			acc.frequencies[str]++
			if len(acc.frequencies) >= StopFreqCountThreshold {
				acc.frequencies = nil
			}
		}

		if len(str) < acc.minLength {
			acc.minLength = len(str)
		}
		if len(str) > acc.maxLength {
			acc.maxLength = len(str)
		}
	}
}

// Map formats stat values as a map
func (acc *stringAcc) Map() map[string]interface{} {
	if acc.count == 0 {
		// avoid reporting default max/min figures, if count is above 0
		// at least one entry has been checked
		return map[string]interface{}{"count": 0}
	}

	m := map[string]interface{}{
		"count":     acc.count,
		"minLength": acc.minLength,
		"maxLength": acc.maxLength,
	}

	if acc.unique != 0 {
		m["unique"] = acc.unique
	}
	if acc.frequencies != nil {
		m["frequencies"] = acc.frequencies
	}

	return m
}

// Close finalizes the accumulator
func (acc *stringAcc) Close() {
	if acc.frequencies != nil {
		// determine unique values
		for key, freq := range acc.frequencies {
			if freq == 1 {
				acc.unique++
				delete(acc.frequencies, key)
			}
		}
		if len(acc.frequencies) == 0 {
			acc.frequencies = nil
		}
	}
}

type boolAcc struct {
	count      int
	trueCount  int
	falseCount int
}

var _ accumulator = (*boolAcc)(nil)

// Type indicates this stat accumulator kind
func (acc *boolAcc) Type() string { return "boolean" }

// Write adds an entry to the stat accumulator
func (acc *boolAcc) Write(e dsio.Entry) {
	if b, ok := e.Value.(bool); ok {
		acc.count++
		if b {
			acc.trueCount++
		} else {
			acc.falseCount++
		}
	}
}

// Map formats stat values as a map
func (acc *boolAcc) Map() map[string]interface{} {
	return map[string]interface{}{
		"count":      acc.count,
		"trueCount":  acc.trueCount,
		"falseCount": acc.falseCount,
	}
}

// Close finalizes the accumulator
func (acc *boolAcc) Close() {}

type nullAcc struct {
	count int
}

var _ accumulator = (*nullAcc)(nil)

// Type indicates this stat accumulator kind
func (acc *nullAcc) Type() string { return "null" }

// Write adds an entry to the stat accumulator
func (acc *nullAcc) Write(e dsio.Entry) {
	if e.Value == nil {
		acc.count++
	}
}

// Map formats stat values as a map
func (acc *nullAcc) Map() map[string]interface{} {
	return map[string]interface{}{"count": acc.count}
}

// Close finalizes the accumulator
func (acc *nullAcc) Close() {}

type keyedStat struct {
	Stat
	key string
}

// Map returns the stat, adding the "key" key indicating which key in the target
// array the stat belongs to
func (ks keyedStat) Map() map[string]interface{} {
	v := ks.Stat.Map()
	v["key"] = ks.key
	return v
}
