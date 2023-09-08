package channelops

import (
	"context"
	"reflect"
	"sync"
)

type ChannelOps interface {
	MergeOrToOne(orChans ...chan any) chan any
	MergeOrToOneIgnoreDuplicates(orChans ...chan any) chan any
}

type channelOps struct {
	lock     *sync.Mutex
	done     chan struct{}
	stopOnce *sync.Once
	doneOnce *sync.Once

	firstCall  bool
	orInterupt chan struct{}
	orChan     chan any

	cancelContextLength int
	selectCases         []reflect.SelectCase
}

// Create a new single use channel operation for managing a combination
// of possible merge stratagies.
func NewChannelOps(cancelContexts ...context.Context) *channelOps {
	orInterupt := make(chan struct{})

	// setup the interupt channel
	selectCases := []reflect.SelectCase{
		{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(orInterupt)},
	}

	// setup the caller cancel channels
	for _, cancelContext := range cancelContexts {
		selectCases = append(selectCases, reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(cancelContext.Done())})
	}

	return &channelOps{
		lock:     new(sync.Mutex),
		done:     make(chan struct{}),
		stopOnce: new(sync.Once),
		doneOnce: new(sync.Once),

		firstCall:  true,
		orInterupt: orInterupt,
		orChan:     make(chan any, 1),

		cancelContextLength: len(cancelContexts),
		selectCases:         selectCases,
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
