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
	stopOnce *sync.Once
	doneOnce *sync.Once

	cancelContext context.Context

	firstCall   bool
	orInterupt  chan struct{}
	orChan      chan any
	selectCases []reflect.SelectCase
}

// Create a new single use channel operation for managing a combination
// of possible merge stratagies.
func NewChannelOps(cancelContext context.Context) *channelOps {
	orInterupt := make(chan struct{})
	selectCases := []reflect.SelectCase{
		{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(orInterupt)},           // we want to interupt
		{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(cancelContext.Done())}, // caller wants to cancel
	}

	return &channelOps{
		lock:     new(sync.Mutex),
		done:     make(chan struct{}),
		stopOnce: new(sync.Once),
		doneOnce: new(sync.Once),

		cancelContext: cancelContext,

		firstCall:   true,
		orInterupt:  orInterupt,
		orChan:      make(chan any, 1),
		selectCases: selectCases,
	}
}

func (co *channelOps) Done() <-chan struct{} {
	return co.done
}

func (co *channelOps) closeDone() {
	co.doneOnce.Do(func() {
		close(co.done)
	})
}

func (co *channelOps) stop() {
	co.stopOnce.Do(func() {
		co.closeDone()
		close(co.orChan)
	})
}
