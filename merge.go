package channelops

import (
	"fmt"
	"reflect"
)

// MergeOrToOne is able to merge any number of provided channels into a single channel
// provided that none of the passed in channels have had a value read from them. At most
// the provided mergeChan will only process one value from any provided orChans.
//
// This function is safe to call asyncronously.
func (co *channelOps) MergeOrToOne(orChans ...chan any) chan any {
	co.lock.Lock()
	switch co.firstCall {
	case true:
		co.firstCall = false
		co.lock.Unlock()
	default:
		co.lock.Unlock()
		select {
		case co.orInterupt <- struct{}{}:
			// try to trigger a stop if a goroutine is already running
		case <-co.done:
			fmt.Println("returning on done")
			// capture race where another thread may have triggered the same time as this call
			return co.orChan
		}
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

	// reeturn the same read channel to the caller every time
	return co.orChan
}

func (co *channelOps) backgroundMergeOrToOne(cases []reflect.SelectCase) {
	index, value, received := reflect.Select(cases)
	switch index {
	case 0:
		// interupted since the caller wants to add more channels
	case 1:
		// the caller cancelled, so close out this 1 time use behavior
		co.stop()
	default:
		// if this immediately recieves, then there is a race where new caller doesn't exit
		if !received {
			// this is a case where the caller closed a channel. We need to remove the closed channel
			co.lock.Lock()
			co.selectCases[index] = co.selectCases[len(co.selectCases)-1] // copy last index into the one we want to drop from being closed
			co.selectCases = co.selectCases[:len(co.selectCases)-1]       // truncate the select cases
			co.lock.Unlock()

			// setup new bacground thread
			go co.backgroundMergeOrToOne(co.selectCases)
		} else {
			co.closeDone()

			// have a value to return the caller
			co.orChan <- value.Interface()
			co.stop()
		}
	}
}
