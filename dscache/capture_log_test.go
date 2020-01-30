package dscache

import (
	"context"
	"fmt"

	golog "github.com/ipfs/go-log"
)

// captureLog can replace this package's logger so that it counts calls instead of actually logging.
// To be used only from tests
type captureLog struct {
	error     int
	errorLast string
	total     int
}

func (l *captureLog) Debug(...interface{}) {
	l.total++
}
func (l *captureLog) Debugf(string, ...interface{}) {
	l.total++
}
func (l *captureLog) Info(...interface{}) {
	l.total++
}
func (l *captureLog) Infof(string, ...interface{}) {
	l.total++
}
func (l *captureLog) LogKV(context.Context, ...interface{}) {
	l.total++
}
func (l *captureLog) Error(...interface{}) {
	l.error++
	l.errorLast = "error"
	l.total++
}
func (l *captureLog) Errorf(tmpl string, vals ...interface{}) {
	l.error++
	l.errorLast = fmt.Sprintf(tmpl, vals...)
	l.total++
}
func (l *captureLog) Event(context.Context, string, ...golog.Loggable) {
}
func (l *captureLog) EventBegin(context.Context, string, ...golog.Loggable) *golog.EventInProgress {
	return nil
}
func (l *captureLog) Start(context.Context, string) context.Context {
	return nil
}
func (l *captureLog) StartFromParentState(context.Context, string, []byte) (context.Context, error) {
	return nil, nil
}
func (l *captureLog) Fatal(...interface{}) {
	l.total++
}
func (l *captureLog) Fatalf(string, ...interface{}) {
	l.total++
}
func (l *captureLog) Panic(...interface{}) {
	l.total++
}
func (l *captureLog) Panicf(string, ...interface{}) {
	l.total++
}
func (l *captureLog) Finish(context.Context) {
}
func (l *captureLog) FinishWithErr(context.Context, error) {
}
func (l *captureLog) SerializeContext(context.Context) ([]byte, error) {
	return nil, nil
}
func (l *captureLog) SetErr(context.Context, error) {
}
func (l *captureLog) SetTag(context.Context, string, interface{}) {
}
func (l *captureLog) SetTags(context.Context, map[string]interface{}) {
}
func (l *captureLog) Warning(...interface{}) {
	l.total++
}
func (l *captureLog) Warningf(string, ...interface{}) {
	l.total++
}

func (l *captureLog) NumErrors() int {
	return l.error
}

func (l *captureLog) ErrorLast() string {
	return l.errorLast
}

func (l *captureLog) NumTotal() int {
	return l.total
}
