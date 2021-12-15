package automation

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/profile"
)

var (
	// ErrEmptyQueue indicates that the queue is empty
	ErrEmptyQueue = fmt.Errorf("empty queue")
)

type runQueueFunc func(context.Context) error

// RunQueue queues runs and apply transforms & allows you to cancel runs and apply transforms
type RunQueue interface {
	Push(ctx context.Context, ownerID string, initID string, runID string, mode string, f runQueueFunc) error
	Pop(ctx context.Context) (*runQueueInfo, error)
	Cancel(runID string) error
	Shutdown() error
}

type runQueueInfo struct {
	ownerID string
	initID  string
	runID   string
	mode    string
	f       runQueueFunc
}

type runQueue struct {
	queue      []*runQueueInfo
	qlk        sync.Mutex
	pub        event.Publisher
	cancels    map[string]context.CancelFunc
	clk        sync.Mutex
	cancelCh   chan string
	closeQueue context.CancelFunc
}

var _ RunQueue = (*runQueue)(nil)

// NewRunQueue returns a RunQueue, that polls every interval to run the next
// run or apply in the queue
// TODO (ramfox): when the run queue's responsibility expands, add an Options struct
func NewRunQueue(ctx context.Context, pub event.Publisher, interval time.Duration, workers int) RunQueue {
	ctx, cancel := context.WithCancel(ctx)

	r := &runQueue{
		queue:      []*runQueueInfo{},
		qlk:        sync.Mutex{},
		pub:        pub,
		cancels:    map[string]context.CancelFunc{},
		clk:        sync.Mutex{},
		cancelCh:   make(chan string),
		closeQueue: cancel,
	}
	if workers == 0 {
		workers = 1
	}
	for i := 0; i < workers; i++ {
		go r.pollQueue(ctx, interval)
	}
	go r.listenForCancelations(ctx)
	return r
}

func (r *runQueue) listenForCancelations(ctx context.Context) {
	log.Debug("listening for run cancelations")
	for {
		select {
		case cancelRunID := <-r.cancelCh:
			r.clk.Lock()
			if _, ok := r.cancels[cancelRunID]; !ok {
				continue
			}
			r.cancels[cancelRunID]()
			r.clk.Unlock()
		case <-ctx.Done():
			return
		}
	}
}

func (r *runQueue) addRunCancel(runID string, cancelFunc context.CancelFunc) {
	r.clk.Lock()
	defer r.clk.Unlock()
	r.cancels[runID] = cancelFunc
}

func (r *runQueue) removeRunCancel(runID string) {
	r.clk.Lock()
	defer r.clk.Unlock()
	delete(r.cancels, runID)
}

func (r *runQueue) pollQueue(ctx context.Context, interval time.Duration) {
	for {
		select {
		case <-time.After(interval):
			info, err := r.Pop(ctx)
			if err != nil {
				continue
			}
			runCtx, cancel := context.WithCancel(ctx)
			r.addRunCancel(info.runID, cancel)
			done := make(chan struct{})
			go func() {
				if err := info.f(runCtx); err != nil {
					log.Debugw("RunQueue", "runID", info.runID, "mode", info.mode, "error", err)
				}
				done <- struct{}{}
			}()
			select {
			case <-done:
				log.Debugw("RunQueue run finished", "runID", info.runID)
			case <-runCtx.Done():
				log.Debugw("RunQueue: context canceled before run finished", "runID", info.runID)
			}
			r.removeRunCancel(info.runID)
		case <-ctx.Done():
			log.Debug("finished polling run queue")
			return
		}
	}
}

func (r *runQueue) Push(ctx context.Context, ownerID string, initID string, runID string, mode string, f runQueueFunc) error {
	r.qlk.Lock()
	defer r.qlk.Unlock()
	scopedCtx := profile.AddIDToContext(ctx, ownerID)
	go func() {
		switch mode {
		case "run":
			if err := r.pub.PublishID(scopedCtx, event.ETAutomationRunQueuePush, runID, &initID); err != nil {
				log.Debug(err)
			}
		case "apply":
			if err := r.pub.PublishID(scopedCtx, event.ETAutomationApplyQueuePush, runID, &initID); err != nil {
				log.Debug(err)
			}
		}
	}()
	info := &runQueueInfo{
		ownerID: ownerID,
		initID:  initID,
		runID:   runID,
		mode:    mode,
		f:       f,
	}
	r.queue = append(r.queue, info)
	return nil
}

func (r *runQueue) Pop(ctx context.Context) (*runQueueInfo, error) {
	r.qlk.Lock()
	defer r.qlk.Unlock()
	if len(r.queue) == 0 {
		return nil, ErrEmptyQueue
	}
	info := r.queue[0]
	r.queue = r.queue[1:]
	go func() {
		switch info.mode {
		case "run":
			if err := r.pub.PublishID(ctx, event.ETAutomationRunQueuePop, info.runID, info.initID); err != nil {
				log.Debug(err)
			}
		case "apply":
			if err := r.pub.Publish(ctx, event.ETAutomationApplyQueuePop, &info.runID); err != nil {
				log.Debug(err)
			}
		}
	}()
	return info, nil
}

func (r *runQueue) Cancel(runID string) error {
	r.cancelCh <- runID
	return nil
}

func (r *runQueue) Shutdown() error {
	r.closeQueue()
	return nil
}
