package remote

import (
	"context"

	"github.com/qri-io/qri/logbook/oplog"
)

// The ctxKey type is unexported to prevent collisions with context keys defined
// in other packages.
type ctxKey int

// oplogKey is the context key for an oplog.Log pointer. log hooks embed oplogs
// in a context for access within a lifecycle hook
const oplogKey ctxKey = 0

// newLogHookContext creates
func newLogHookContext(ctx context.Context, l *oplog.Log) context.Context {
	return context.WithValue(ctx, oplogKey, l)
}

// OplogFromContext pulls an oplog value from
func OplogFromContext(ctx context.Context) (l *oplog.Log, ok bool) {
	l, ok = ctx.Value(oplogKey).(*oplog.Log)
	return l, ok
}