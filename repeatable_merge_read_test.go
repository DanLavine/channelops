package channelops

import (
	"context"
	"testing"
	"time"

	. "github.com/onsi/gomega"
)

func Test_RepeatableMergeRead_SendingDefaultTypes(t *testing.T) {
	g := NewGomegaWithT(t)

	t.Run("It works with interface{}", func(t *testing.T) {
		channelOps, reader := NewRepeatableMergeRead[any](false, context.Background())

		chanOne := make(chan any)
		go func() {
			chanOne <- "one"
		}()

		err := channelOps.MergeOrToOne(chanOne)
		g.Expect(err).ToNot(HaveOccurred())

		select {
		case value := <-reader:
			g.Expect(value.Value).To(Equal("one"))
			g.Expect(value.Continue).ToNot(BeNil())
			g.Expect(value.Stop).ToNot(BeNil())

			value.Stop()
		case <-time.After(time.Second):
			g.Fail("failed to pull an item from the reader")
		}
	})

	t.Run("It works with struct{}", func(t *testing.T) {
		channelOps, reader := NewRepeatableMergeRead[struct{}](true, context.Background())

		chanOne := make(chan struct{})
		go func() {
			chanOne <- struct{}{}
		}()

		err := channelOps.MergeOrToOne(chanOne)
		g.Expect(err).ToNot(HaveOccurred())

		select {
		case value := <-reader:
			g.Expect(value.Value).To(Equal(struct{}{}))
			g.Expect(value.Continue).ToNot(BeNil())
			g.Expect(value.Stop).ToNot(BeNil())

			value.Stop()
		case <-time.After(time.Second):
			g.Fail("failed to pull an item from the reader")
		}
	})

	t.Run("It works with string", func(t *testing.T) {
		channelOps, reader := NewRepeatableMergeRead[string](true, context.Background())

		chanOne := make(chan string)
		go func() {
			chanOne <- "other"
		}()

		err := channelOps.MergeOrToOne(chanOne)
		g.Expect(err).ToNot(HaveOccurred())

		select {
		case value := <-reader:
			g.Expect(value.Value).To(Equal("other"))
			g.Expect(value.Continue).ToNot(BeNil())
			g.Expect(value.Stop).ToNot(BeNil())

			value.Stop()
		case <-time.After(time.Second):
			g.Fail("failed to pull an item from the reader")
		}
	})

	t.Run("It works with int", func(t *testing.T) {
		channelOps, reader := NewRepeatableMergeRead[int](true, context.Background())

		chanOne := make(chan int)
		go func() {
			chanOne <- 3
		}()

		err := channelOps.MergeOrToOne(chanOne)
		g.Expect(err).ToNot(HaveOccurred())

		select {
		case value := <-reader:
			g.Expect(value.Value).To(Equal(3))
			g.Expect(value.Continue).ToNot(BeNil())
			g.Expect(value.Stop).ToNot(BeNil())

			value.Stop()
		case <-time.After(time.Second):
			g.Fail("failed to pull an item from the reader")
		}
	})
}

func Test_RepeatableMergeRead_Repeating(t *testing.T) {
	g := NewGomegaWithT(t)

	t.Run("It continues to process if Continue is called", func(t *testing.T) {
		channelOps, reader := NewRepeatableMergeRead[any](false, context.Background())

		chanOne := make(chan any)
		go func() {
			chanOne <- "one"
			chanOne <- "two"
		}()

		err := channelOps.MergeOrToOne(chanOne)
		g.Expect(err).ToNot(HaveOccurred())

		// read the first item
		select {
		case value := <-reader:
			g.Expect(value.Value).To(Equal("one"))
			g.Expect(value.Continue).ToNot(BeNil())
			g.Expect(value.Stop).ToNot(BeNil())

			value.Continue()
		case <-time.After(time.Second):
			g.Fail("failed to pull an item from the reader")
		}

		// read the second item
		select {
		case value := <-reader:
			g.Expect(value.Value).To(Equal("two"))
			g.Expect(value.Continue).ToNot(BeNil())
			g.Expect(value.Stop).ToNot(BeNil())

			value.Stop()
		case <-time.After(time.Second):
			g.Fail("failed to pull an item from the reader")
		}
	})

	t.Run("It stops processing if Stop is called", func(t *testing.T) {
		channelOps, reader := NewRepeatableMergeRead[any](false, context.Background())

		chanOne := make(chan any, 3)
		go func() {
			chanOne <- "one"
			chanOne <- "two"
		}()

		err := channelOps.MergeOrToOne(chanOne)
		g.Expect(err).ToNot(HaveOccurred())

		// read the first item
		select {
		case value := <-reader:
			g.Expect(value.Value).To(Equal("one"))
			g.Expect(value.Continue).ToNot(BeNil())
			g.Expect(value.Stop).ToNot(BeNil())

			value.Stop()
		case <-time.After(time.Second):
			g.Fail("failed to pull an item from the reader")
		}

		g.Consistently(reader).ShouldNot(Receive())
		g.Eventually(reader).Should(BeClosed())

	})
}

func Test_RepeatableMergeRead_MergeOrToOne(t *testing.T) {
	g := NewGomegaWithT(t)

	t.Run("It can read from any of the channels merged", func(t *testing.T) {
		channelOps, reader := NewRepeatableMergeRead[any](false, context.Background())

		chanOne := make(chan any)
		chanTwo := make(chan any)

		go func() {
			chanOne <- "one"
		}()

		go func() {
			chanTwo <- "two"
		}()

		// ensure that whatever value we read in the merge, we can obtain the other value.
		err := channelOps.MergeOrToOne(chanOne, chanTwo)
		g.Expect(err).ToNot(HaveOccurred())

		value := <-reader
		switch value.Value {
		case "one":
			g.Eventually(chanTwo).Should(Receive(Equal("two")))
		case "two":
			g.Eventually(chanOne).Should(Receive(Equal("one")))
		}

		value.Stop()
		g.Eventually(reader).Should(BeClosed())
		g.Eventually(channelOps.Done()).Should(BeClosed())
	})

	t.Run("It allows for passed in channels to be clossed without iterupting the OR selection", func(t *testing.T) {
		channelOps, reader := NewRepeatableMergeRead[any](false, context.Background())

		chanOne := make(chan any)
		chanTwo := make(chan any)

		channelOps.MergeOrToOne(chanOne, chanTwo)

		close(chanOne)
		go func() {
			chanTwo <- "two"
		}()

		select {
		case value := <-reader:
			g.Expect(value.Value).To(Equal("two"))
			value.Stop()
		case <-time.After(time.Second):
			g.Fail("failed to pull an item from the reader")
		}
		g.Eventually(channelOps.Done()).Should(BeClosed())
	})

	t.Run("It allows for passed in channels to be clossed and stopping the OR selection", func(t *testing.T) {
		channelOps, reader := NewRepeatableMergeRead[any](true, context.Background())

		chanOne := make(chan any)
		chanTwo := make(chan any)

		channelOps.MergeOrToOne(chanOne, chanTwo)

		close(chanOne)
		g.Eventually(reader).Should(BeClosed())
		g.Eventually(channelOps.Done()).Should(BeClosed())
	})

	t.Run("It allows for a channel to receive nil", func(t *testing.T) {
		channelOps, reader := NewRepeatableMergeRead[any](false, context.Background())

		chanOne := make(chan any)
		chanTwo := make(chan any)

		channelOps.MergeOrToOne(chanOne, chanTwo)

		go func() {
			chanOne <- nil
		}()

		select {
		case value := <-reader:
			g.Expect(value.Value).To(BeNil())
			value.Stop()
		case <-time.After(time.Second):
			g.Fail("failed to pull an item from the reader")
		}
		g.Eventually(channelOps.Done()).Should(BeClosed())
	})

	t.Run("It cloeses the merged channel if the context is closed", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		channelOps, reader := NewRepeatableMergeRead[any](false, ctx)

		chanOne := make(chan any)
		chanTwo := make(chan any)

		channelOps.MergeOrToOne(chanOne, chanTwo)

		go func() {
			cancel()
		}()

		g.Eventually(reader).Should(BeClosed())
		g.Eventually(channelOps.Done()).Should(BeClosed())
	})

	t.Run("It works with a large number of processes in parallel", func(t *testing.T) {
		counter := 10_000

		ctx, cancel := context.WithCancel(context.Background())
		channelOps, reader := NewRepeatableMergeRead[int](false, ctx)

		// create a bunch of channels and write to them
		for i := 0; i < counter; i++ {
			go func(ii int) {
				channel := make(chan int)
				_ = channelOps.MergeOrToOne(channel)

				go func(num int, numChan chan int) {
					numChan <- num
					close(numChan) //cleanup the channel. without doing this, the tests are slow
				}(ii, channel)
			}(i)
		}

		foundItems := []int{}
		for i := 0; i < counter; i++ {
			select {
			case value := <-reader:
				g.Expect(foundItems).ToNot(ContainElement(value.Value))

				foundItems = append(foundItems, value.Value)
				value.Continue()
			case <-time.After(20 * time.Second):
				g.Fail("failed to pull an item from the reader")
			}
		}

		cancel()
		g.Eventually(reader).Should(BeClosed())
		g.Eventually(channelOps.Done()).Should(BeClosed())
	})

	t.Run("It returns an error if the channel has processed or a cancel contet was closed", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		channelOps, reader := NewRepeatableMergeRead[any](false, ctx)
		g.Eventually(reader).Should(BeClosed())
		g.Eventually(channelOps.Done()).Should(BeClosed())

		chanOne := make(chan any)
		err := channelOps.MergeOrToOne(chanOne)
		g.Expect(err).ToNot(BeNil())
		g.Expect(err.Error()).To(Equal("channel has stopped processing"))
	})
}

func Test_RepeatableMergeRead_MergeOrToOneIgnoreDuplicates(t *testing.T) {
	g := NewGomegaWithT(t)

	t.Run("It can read from any of the channels merged", func(t *testing.T) {
		channelOps, reader := NewRepeatableMergeRead[any](false, context.Background())

		chanOne := make(chan any)
		chanTwo := make(chan any)

		go func() {
			chanOne <- "one"
		}()

		go func() {
			chanTwo <- "two"
		}()

		// ensure that whatever value we read in the merge, we can obtain the other value.
		err := channelOps.MergeOrToOneIgnoreDuplicates(chanOne, chanTwo)
		g.Expect(err).ToNot(HaveOccurred())

		value := <-reader
		switch value.Value {
		case "one":
			g.Eventually(chanTwo).Should(Receive(Equal("two")))
		case "two":
			g.Eventually(chanOne).Should(Receive(Equal("one")))
		}

		value.Stop()
		g.Eventually(reader).Should(BeClosed())
		g.Eventually(channelOps.Done()).Should(BeClosed())
	})

	t.Run("It only adds the channel a single time on multiple calls", func(t *testing.T) {
		channelOps, reader := NewRepeatableMergeRead[any](false, context.Background())
		chanOne := make(chan any)

		// ensure that whatever value we read in the merge, we can obtain the other value.
		g.Expect(channelOps.MergeOrToOneIgnoreDuplicates(chanOne)).ToNot(HaveOccurred())
		g.Expect(len(channelOps.selectCases)).To(Equal(3))

		g.Expect(channelOps.MergeOrToOneIgnoreDuplicates(chanOne)).ToNot(HaveOccurred())
		g.Expect(channelOps.MergeOrToOneIgnoreDuplicates(chanOne)).ToNot(HaveOccurred())
		g.Expect(channelOps.MergeOrToOneIgnoreDuplicates(chanOne)).ToNot(HaveOccurred())
		g.Expect(channelOps.MergeOrToOneIgnoreDuplicates(chanOne)).ToNot(HaveOccurred())
		g.Expect(len(channelOps.selectCases)).To(Equal(3))

		go func() {
			chanOne <- "one"
		}()

		select {
		case value := <-reader:
			g.Expect(value.Value).To(Equal("one"))
			value.Stop()
		case <-time.After(time.Second):
			g.Fail("failed to pull an item from the reader")
		}
		g.Eventually(channelOps.Done()).Should(BeClosed())
	})

	t.Run("It allows for passed in channels to be clossed without iterupting the OR selection", func(t *testing.T) {
		channelOps, reader := NewRepeatableMergeRead[any](false, context.Background())

		chanOne := make(chan any)
		chanTwo := make(chan any)

		err := channelOps.MergeOrToOneIgnoreDuplicates(chanOne, chanTwo)
		g.Expect(err).ToNot(HaveOccurred())

		close(chanOne)
		go func() {
			chanTwo <- "two"
		}()

		select {
		case value := <-reader:
			g.Expect(value.Value).To(Equal("two"))
			value.Stop()
		case <-time.After(time.Second):
			g.Fail("failed to pull an item from the reader")
		}
		g.Eventually(channelOps.Done()).Should(BeClosed())
	})

	t.Run("It allows for passed in channels to be clossed and stopping the OR selection", func(t *testing.T) {
		channelOps, reader := NewRepeatableMergeRead[any](true, context.Background())

		chanOne := make(chan any)
		chanTwo := make(chan any)

		err := channelOps.MergeOrToOneIgnoreDuplicates(chanOne, chanTwo)
		g.Expect(err).ToNot(HaveOccurred())

		close(chanOne)
		g.Eventually(reader).Should(BeClosed())
		g.Eventually(channelOps.Done()).Should(BeClosed())
	})

	t.Run("It allows for a channel to receive nil", func(t *testing.T) {
		channelOps, reader := NewRepeatableMergeRead[any](false, context.Background())

		chanOne := make(chan any)
		chanTwo := make(chan any)

		err := channelOps.MergeOrToOneIgnoreDuplicates(chanOne, chanTwo)
		g.Expect(err).ToNot(HaveOccurred())

		go func() {
			chanOne <- nil
		}()

		select {
		case value := <-reader:
			g.Expect(value.Value).To(BeNil())
			value.Stop()
		case <-time.After(time.Second):
			g.Fail("failed to pull an item from the reader")
		}
		g.Eventually(channelOps.Done()).Should(BeClosed())
	})

	t.Run("It cloeses the merged channel if the context is closed", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		channelOps, reader := NewRepeatableMergeRead[any](false, ctx)

		chanOne := make(chan any)
		chanTwo := make(chan any)

		err := channelOps.MergeOrToOneIgnoreDuplicates(chanOne, chanTwo)
		g.Expect(err).ToNot(HaveOccurred())

		go func() {
			cancel()
		}()

		g.Eventually(reader).Should(BeClosed())
		g.Eventually(channelOps.Done()).Should(BeClosed())
	})

	t.Run("It works with a large number of processes in parallel", func(t *testing.T) {
		counter := 10_000

		ctx, cancel := context.WithCancel(context.Background())
		channelOps, reader := NewRepeatableMergeRead[int](false, ctx)

		// create a bunch of channels and write to them
		for i := 0; i < counter; i++ {
			go func(ii int) {
				channel := make(chan int)
				_ = channelOps.MergeOrToOneIgnoreDuplicates(channel)

				go func(num int, numChan chan int) {
					numChan <- num
					close(numChan) //cleanup the channel. without doing this, the tests are slow
				}(ii, channel)
			}(i)
		}

		foundItems := []int{}
		for i := 0; i < counter; i++ {
			select {
			case value := <-reader:
				g.Expect(foundItems).ToNot(ContainElement(value.Value))

				foundItems = append(foundItems, value.Value)
				value.Continue()
			case <-time.After(20 * time.Second):
				g.Fail("failed to pull an item from the reader")
			}
		}

		cancel()
		g.Eventually(reader).Should(BeClosed())
		g.Eventually(channelOps.Done()).Should(BeClosed())
	})

	t.Run("It returns an error if the channel has processed or a cancel contet was closed", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		channelOps, reader := NewRepeatableMergeRead[any](false, ctx)
		g.Eventually(reader).Should(BeClosed())
		g.Eventually(channelOps.Done()).Should(BeClosed())

		chanOne := make(chan any)
		err := channelOps.MergeOrToOneIgnoreDuplicates(chanOne)
		g.Expect(err).ToNot(BeNil())
		g.Expect(err.Error()).To(Equal("channel has stopped processing"))
	})
}
