package channelops

import (
	"reflect"
)

// MergeOrToOne is able to merge any number of provided channels into a single channel
// provided that none of the passed in channels have had a value read from them. At most
// the provided mergeChan will only process one value from any provided orChans.
//
// This function is safe to call asyncronously.
func (co *channelOps) MergeOrToOne(orChans ...chan any) chan any {
	select {
	case co.orInterupt <- struct{}{}:
		// try to trigger a stop if a goroutine is already running
	case <-co.done:
		// capture race where another thread may have triggered the same time as this call
		return co.orChan
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

// MergeOrToOneIgnoreDuplicates is the same as MerOrToOne, but explicitly checks to make sure that all passed in
// channels are not already being read from. Any that are will be ignored
func (co *channelOps) MergeOrToOneIgnoreDuplicates(orChans ...chan any) chan any {
	select {
	case co.orInterupt <- struct{}{}:
		// try to trigger a stop if a goroutine is already running
	case <-co.done:
		// capture race where another thread may have triggered the same time as this call
		return co.orChan
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

	// reeturn the same read channel to the caller every time
	return co.orChan
}

func (co *channelOps) backgroundMergeOrToOne(cases []reflect.SelectCase) {
	index, value, received := reflect.Select(cases)
	if index == 0 {
		// interupted since the caller wants to add more channels
	} else if index <= co.cancelContextLength {
		// the caller cancelled, so close out this 1 time use behavior
		co.stop()
	} else {
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
