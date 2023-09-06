package channelops

import (
	"reflect"
)

// MergeOrToOne is able to merge any number of provided channels into a single channel
// provided that none of the channels have had a value read from them. At most the provided
// mergeChan will only process one value from any provided orChans.
//
// In addition to this, as long as the merge chan referance is kept, the orChans can all be
// added asynchronously
func (co *channelOps) MergeOrToOne(orChans ...chan any) chan any {
	co.lock.Lock()
	defer co.lock.Unlock()

	select {
	case <-co.done:
		// channel already returned something, so bail since this is a 1 time use
		return co.orChan
	default:
		// fall through to the main logic
	}

	if co.orInterupt != nil {
		// must already have a thread running in the background

		// cancel the running thread
		co.orInterupt <- struct{}{}

		// wait for a response to know the background thread was canceled
		select {
		case <-co.done:
			// capture race where a background thread may have triggered the same time as this call
			return co.orChan
		case <-co.orInterupt:
			// fall through and setup the select cases again
		}
	} else {
		co.selectCases = []reflect.SelectCase{
			{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(co.orInterupt)},           // we want to interupt
			{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(co.cancelContext.Done())}, // caller wants to cancel
		}
		co.orInterupt = make(chan struct{})
	}

	// add all provided select cases
	for _, orChan := range orChans {
		co.selectCases = append(co.selectCases, reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(orChan)})
	}

	go func() {
		index, value, _ := reflect.Select(co.selectCases)
		switch index {
		case 0:
			co.orInterupt <- struct{}{}
		case 1:
			// the caller canceld, so close out this 1 time use behavior
			co.stop()
		default:
			co.orChan <- value.Interface()
			co.stop()
		}
	}()

	return co.orChan
}
