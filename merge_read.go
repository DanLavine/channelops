package channelops

import (
	"context"
	"fmt"
	"reflect"
	"sync"
)

type mergeReadChannelOps[T any] struct {
	lock     *sync.Mutex
	done     chan struct{}
	stopOnce *sync.Once
	doneOnce *sync.Once

	stopOnClose bool
	orInterupt  chan struct{}
	orChan      chan T

	cancelContextLength int
	selectCases         []reflect.SelectCase
}

// Create a new single use channel operation for merging write channels. All functions
// for this data type are thread safe to call asynchronously
//
//	PARAMETERS:
//	- stopOnClose -Iff TRUE, any merged channels that are closed trigger the '<-chan T' ot be closed
//	- cancelContexts - any contexts when canceled will close '<-chan T'
//
// Known limitations:
//
// 1. Only 65535 channels can be added to a single merge strategy. (There is a way to increase this, but untill I have an actual use case for that I think its fine)
func NewMergeRead[T any](stopOnClose bool, cancelContexts ...context.Context) (MergeReadChannelOps[T], <-chan T) {
	orInterupt := make(chan struct{})

	// setup the interupt channel
	selectCases := []reflect.SelectCase{
		{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(orInterupt)},
	}

	// setup the caller cancel channels
	for _, cancelContext := range cancelContexts {
		selectCases = append(selectCases, reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(cancelContext.Done())})
	}

	orChan := make(chan T, 1)
	channelOps := &mergeReadChannelOps[T]{
		lock:     new(sync.Mutex),
		done:     make(chan struct{}),
		stopOnce: new(sync.Once),
		doneOnce: new(sync.Once),

		stopOnClose: stopOnClose,
		orInterupt:  orInterupt,
		orChan:      orChan,

		cancelContextLength: len(cancelContexts),
		selectCases:         selectCases,
	}

	go channelOps.backgroundMergeOrToOne(selectCases)

	return channelOps, orChan
}

func (co *mergeReadChannelOps[T]) Done() <-chan struct{} {
	return co.done
}

// MergeOrToOne is able to merge any number of provided channels into a single channel
// provided that none of the passed in channels have had a value read from them. At most
// the provided mergeChan will only process one value from any provided orChans.
//
// This function is safe to call asyncronously.
func (co *mergeReadChannelOps[T]) MergeOrToOne(orChans ...<-chan T) error {
	select {
	case co.orInterupt <- struct{}{}:
		// try to trigger a stop if a goroutine is already running
	case <-co.done:
		// capture race where another thread may have triggered the same time as this call
		return fmt.Errorf("channel has already processed a read operation")
	}

	// add all provided select cases
	co.lock.Lock()
	for _, orChan := range orChans {
		co.selectCases = append(co.selectCases, reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(orChan)})
	}
	cases := co.selectCases
	co.lock.Unlock()

	// start the multiplexer in the background
	go co.backgroundMergeOrToOne(cases)

	return nil
}

// MergeOrToOneIgnoreDuplicates is the same as MerOrToOne, but explicitly checks to make sure that all passed in
// channels are not already being read from. Any that are will be ignored
func (co *mergeReadChannelOps[T]) MergeOrToOneIgnoreDuplicates(orChans ...<-chan T) error {
	select {
	case co.orInterupt <- struct{}{}:
		// try to trigger a stop if a goroutine is already running
	case <-co.done:
		// capture race where another thread may have triggered the same time as this call
		return fmt.Errorf("channel has already processed a read operation")
	}

	// add all provided select cases
	co.lock.Lock()
	for _, orChan := range orChans {
		add := true

		// loop through the know cases and drop any that are already known
		for _, knownCase := range co.selectCases {
			if knownCase.Chan.Interface() == orChan {
				add = false
				break
			}
		}

		if add {
			co.selectCases = append(co.selectCases, reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(orChan)})
		}
	}
	cases := co.selectCases
	co.lock.Unlock()

	// start the multiplexer in the background
	go co.backgroundMergeOrToOne(cases)

	return nil
}

func (co *mergeReadChannelOps[T]) backgroundMergeOrToOne(cases []reflect.SelectCase) {
	index, value, received := reflect.Select(cases)
	if index == 0 {
		// interupted since the caller wants to add more channels
	} else if index <= co.cancelContextLength {
		// the caller cancelled, so close out this 1 time use behavior
		co.stop()
	} else {
		// if this immediately recieves, then there is a race where new caller doesn't exit
		if !received && !co.stopOnClose {
			// this is a case where the caller closed a channel. We need to remove the closed channel
			co.lock.Lock()
			co.selectCases[index] = co.selectCases[len(co.selectCases)-1] // copy last index into the one we want to drop from being closed
			co.selectCases = co.selectCases[:len(co.selectCases)-1]       // truncate the select cases
			co.lock.Unlock()

			// setup new bacground thread
			go co.backgroundMergeOrToOne(co.selectCases)
		} else if !received {
			// stopped because a channel was closed
			co.stop()
		} else {
			co.closeDone()

			// have a value to return the caller
			if value.Interface() == nil {
				// nothing to do here. the caller will be closed in the nil case
				var empty T
				co.orChan <- empty
			} else {
				co.orChan <- value.Interface().(T) // cast to the type of channel we are
			}
			co.stop()
		}
	}
}

func (co *mergeReadChannelOps[T]) closeDone() {
	co.doneOnce.Do(func() {
		close(co.done)
	})
}

func (co *mergeReadChannelOps[T]) stop() {
	co.stopOnce.Do(func() {
		co.closeDone()
		close(co.orChan)
	})
}
