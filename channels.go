package channelops

import (
	"context"
	"reflect"
	"sync"
)

type ChannelOps interface {
	MergeOrToOne(orChans ...chan any)
}

type channelOps struct {
	lock     *sync.Mutex
	done     chan struct{}
	doneOnce *sync.Once

	cancelContext context.Context

	orInterupt  chan struct{}
	orChan      chan any
	selectCases []reflect.SelectCase
}

// Create a new single use channel operation for managing a combination
// of possible merge stratagies.
func NewChannelOps(cancelContext context.Context) *channelOps {
	return &channelOps{
		lock:     new(sync.Mutex),
		done:     make(chan struct{}),
		doneOnce: new(sync.Once),

		cancelContext: cancelContext,

		orChan: make(chan any, 1),
	}
}

func (co *channelOps) Done() <-chan struct{} {
	return co.done
}

func (co *channelOps) stop() {
	co.doneOnce.Do(func() {
		co.lock.Lock()
		defer co.lock.Unlock()

		if co.orInterupt != nil {
			close(co.orInterupt)
		}
		close(co.done)
		close(co.orChan)
	})
}
