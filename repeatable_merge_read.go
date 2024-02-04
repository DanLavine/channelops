package channelops

import (
	"context"
	"fmt"
	"reflect"
	"sync"
)

// Repeater is the value returned from any channel read operation. A call to `Continue` or `Stop`
// must be called to either allow for another read or cleanup resources
type Reapeater[T any] struct {
	// Value that was read from the channel
	Value T

	// Continue is used to inform the reapeater that it should continue to process read operations
	Continue func()
	// Stop is ues to inform the repeater that it no longer should process read operations
	Stop func()
}

type RepeatableMergeReadChannelOps[T any] struct {
	lock     *sync.Mutex
	done     chan struct{}
	stopOnce *sync.Once

	stopOnClose bool
	orInterupt  chan struct{}
	orChan      chan Reapeater[T]

	continueProcessing chan struct{}
	stopProcessing     chan struct{}

	cancelContextLength int
	selectCases         []reflect.SelectCase
}

//	PARAMETERS:
//	- stopOnClose -Iff TRUE, any merged channels that are closed trigger the '<-chan Reapeater[T]' to be closed
//	- cancelContexts - any contexts when canceled will close '<-chan Reapeater[T]'
//
//	RETURNS:
//	- RepeatableMergeReadChannelOps[T] - channel op that additional readers can be added to.
//	- <-chan Reapeater[T] - struct that contains the value read from the channel. As well as the Continue of Stop operations for the channel op
//
// Create a new multiple use channel operation for merging any number of read channels into a single read operation.
// To continue reading after after “<-chan Reapeater[T]` returns a value, `Repeater.Continue()` must be called. Similarly
// to stop reading `Repeater.Stop()` must be called to release any resources held by this channel operator
//
// Known limitations:
// 1. Only (65535 - N context) channels can be added to a single merge strategy. (There is a way to increase this, with a timer. But untill I have an actual use case for that I think it is fine)
func NewRepeatableMergeRead[T any](stopOnClose bool, cancelContexts ...context.Context) (*RepeatableMergeReadChannelOps[T], <-chan Reapeater[T]) {
	orInterupt := make(chan struct{})

	// setup the interupt channel
	selectCases := []reflect.SelectCase{
		{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(orInterupt)},
	}

	// setup the caller cancel channels
	for _, cancelContext := range cancelContexts {
		selectCases = append(selectCases, reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(cancelContext.Done())})
	}

	orChan := make(chan Reapeater[T], 1)
	channelOps := &RepeatableMergeReadChannelOps[T]{
		lock:     new(sync.Mutex),
		done:     make(chan struct{}),
		stopOnce: new(sync.Once),

		stopOnClose: stopOnClose,
		orInterupt:  orInterupt,
		orChan:      orChan,

		continueProcessing: make(chan struct{}),
		stopProcessing:     make(chan struct{}),

		cancelContextLength: len(cancelContexts),
		selectCases:         selectCases,
	}

	go channelOps.backgroundMergeOrToOne(selectCases)

	return channelOps, orChan
}

// RETURNS:
// - <-chan struct{} - channel to indicate if this merger read operation has processed and will no longer process
//
// Done can be used to detemerine if the channel has been read from and will no longer process anymore read operations
func (co *RepeatableMergeReadChannelOps[T]) Done() <-chan struct{} {
	return co.done
}

//	PARAMETERS:
//	- orChans - any number of channels to merge into the one reader
//
//	RETURNS:
//	- error - error if the reader has already responded and stopped proccessing
//
// MergeOrToOne is able to merge any number of provided channels into the single channel
// provided that none of the passed in channels have had a value read from them.
//
// This function is safe to call asyncronously.
func (co *RepeatableMergeReadChannelOps[T]) MergeOrToOne(orChans ...<-chan T) error {
	select {
	case co.orInterupt <- struct{}{}:
		// try to trigger a stop if a goroutine is already running
	case <-co.done:
		// capture race where another thread may have triggered the same time as this call
		return fmt.Errorf("channel has stopped processing")
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

//	PARAMETERS:
//	- orChans - any number of channels to merge into the one reader
//
//	RETURNS:
//	- error - error if the reader has already responded and stopped proccessing
//
// MergeOrToOneIgnoreDuplicates is able to merge any number of provided channels into the single channel
// provided that none of the passed in channels have had a value read from them. When adding channels,
// the previous channels are compared to ensure that the same channel is not added multiple times.
//
// This function is safe to call asyncronously.
func (co *RepeatableMergeReadChannelOps[T]) MergeOrToOneIgnoreDuplicates(orChans ...<-chan T) error {
	select {
	case co.orInterupt <- struct{}{}:
		// try to trigger a stop if a goroutine is already running
	case <-co.done:
		// capture race where another thread may have triggered the same time as this call
		return fmt.Errorf("channel has stopped processing")
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

func (co *RepeatableMergeReadChannelOps[T]) backgroundMergeOrToOne(cases []reflect.SelectCase) {
	index, value, received := reflect.Select(cases)
	if index == 0 {
		// interupted since the caller wants to add more channels
	} else if index <= co.cancelContextLength {
		// the caller cancelled, so close out this 1 time use behavior
		co.stop()
	} else {
		if !received && !co.stopOnClose {
			// this is a case where a reader channel was closed and we need to remove the closed channel, but allow for another channel to process
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
			// have a value to return the caller
			if value.Interface() == nil {
				// in this case, we read a nil value. so set that and let the channel reader decide to continue or not
				var empty T
				co.orChan <- Reapeater[T]{
					Value:    empty,
					Continue: co.next,
					Stop:     co.halt,
				}
			} else {
				co.orChan <- Reapeater[T]{
					Value:    value.Interface().(T), // cast to the type of channel we are
					Continue: co.next,
					Stop:     co.halt,
				}
			}

			// need to wait for the caller to chose to continue processing or halt
			select {
			case <-co.continueProcessing:
				// setup new bacground thread
				go co.backgroundMergeOrToOne(co.selectCases)
			case <-co.stopProcessing:
				co.stop()
			}
		}
	}
}

func (co *RepeatableMergeReadChannelOps[T]) next() {
	co.continueProcessing <- struct{}{}
}

func (co *RepeatableMergeReadChannelOps[T]) halt() {
	co.stopProcessing <- struct{}{}
}

func (co *RepeatableMergeReadChannelOps[T]) stop() {
	co.stopOnce.Do(func() {
		close(co.done)
		close(co.orChan)
	})
}
