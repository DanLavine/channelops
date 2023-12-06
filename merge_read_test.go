package channelops

import (
	"context"
	"sync"
	"testing"

	. "github.com/onsi/gomega"
)

func Test_MergeOrToOne(t *testing.T) {
	g := NewGomegaWithT(t)

	t.Run("it ensures only one possible channel is read from", func(t *testing.T) {
		channelOps, reader := NewMergeRead[any](context.Background())

		chanOne := make(chan any)
		chanTwo := make(chan any)

		go func() {
			chanOne <- "one"
		}()

		go func() {
			chanTwo <- "two"
		}()

		// ensure that whatever value we read in the merge, we can obtain the other value.
		channelOps.MergeOrToOne(chanOne, chanTwo)
		value := <-reader
		switch value.(string) {
		case "one":
			g.Eventually(chanTwo).Should(Receive(Equal("two")))
		case "two":
			g.Eventually(chanOne).Should(Receive(Equal("one")))
		}

		g.Expect(reader).To(BeClosed())
	})

	t.Run("it allows for passed in channels to be clossed without iterupting the OR selection", func(t *testing.T) {
		channelOps, reader := NewMergeRead[any](context.Background())

		chanOne := make(chan any)
		chanTwo := make(chan any)

		channelOps.MergeOrToOne(chanOne, chanTwo)

		close(chanOne)
		go func() {
			chanTwo <- "two"
		}()

		g.Eventually(reader).Should(Receive(Equal("two")))
	})

	t.Run("it properly exits if a channel recieved nil", func(t *testing.T) {
		channelOps, reader := NewMergeRead[any](context.Background())

		chanOne := make(chan any)
		chanTwo := make(chan any)

		channelOps.MergeOrToOne(chanOne, chanTwo)

		go func() {
			chanOne <- nil
		}()

		g.Eventually(reader).Should(Receive(BeNil()))
		g.Eventually(reader).Should(BeClosed())
		g.Expect(channelOps.Done()).To(BeClosed())
	})

	t.Run("it cloeses the merged channel if the context is closed", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		channelOps, reader := NewMergeRead[any](ctx)

		chanOne := make(chan any)
		chanTwo := make(chan any)

		channelOps.MergeOrToOne(chanOne, chanTwo)

		go func() {
			cancel()
		}()

		g.Eventually(reader).ShouldNot(Receive())
		g.Eventually(reader).Should(BeClosed())
		g.Expect(channelOps.Done()).To(BeClosed())
	})

	t.Run("it works with a large number of processes in parallel", func(t *testing.T) {
		done := make(chan struct{})
		wg := new(sync.WaitGroup)
		counter := 10_000
		channelOps, reader := NewMergeRead[any](context.Background())

		channelOps.MergeOrToOne(nil)

		for i := 0; i < counter; i++ {
			wg.Add(1)
			go func(ii int) {
				defer wg.Done()
				channel := make(chan any)
				_ = channelOps.MergeOrToOne(channel)

				go func(num int, numChan chan any) {
					select {
					case numChan <- num:
					case <-done:
					}
				}(ii, channel)
			}(i)
		}

		wg.Wait()

		g.Eventually(reader).Should(Receive(And(BeNumerically(">=", 0), BeNumerically("<", counter))))
		g.Eventually(reader).Should(BeClosed())
	})

	t.Run("it returns an error if the channel has processed or a cancel contet was closed", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		channelOps, reader := NewMergeRead[any](ctx)
		g.Eventually(reader).Should(BeClosed())

		chanOne := make(chan any)
		err := channelOps.MergeOrToOne(chanOne)
		g.Expect(err).ToNot(BeNil())
		g.Expect(err.Error()).To(Equal("channel has already processed a read operation"))
	})
}

func Test_MergeOrToOneIgnoreDuplicates(t *testing.T) {
	g := NewGomegaWithT(t)

	t.Run("it ensures only one possible channel is read from", func(t *testing.T) {
		channelOps, reader := NewMergeRead[any](context.Background())

		chanOne := make(chan any)
		chanTwo := make(chan any)

		go func() {
			chanOne <- "one"
		}()

		go func() {
			chanTwo <- "two"
		}()

		// ensure that whatever value we read in the merge, we can obtain the other value.
		channelOps.MergeOrToOneIgnoreDuplicates(chanOne, chanTwo)
		value := <-reader
		switch value.(string) {
		case "one":
			g.Eventually(chanTwo).Should(Receive(Equal("two")))
		case "two":
			g.Eventually(chanOne).Should(Receive(Equal("one")))
		}

		g.Expect(reader).To(BeClosed())
	})

	t.Run("it allows for passed in channels to be clossed without iterupting the OR selection", func(t *testing.T) {
		channelOps, reader := NewMergeRead[any](context.Background())

		chanOne := make(chan any)
		chanTwo := make(chan any)

		channelOps.MergeOrToOneIgnoreDuplicates(chanOne, chanTwo)

		close(chanOne)
		go func() {
			chanTwo <- "two"
		}()

		g.Eventually(reader).Should(Receive(Equal("two")))
	})

	t.Run("it properly exits if a channel recieved nil", func(t *testing.T) {
		channelOps, reader := NewMergeRead[any](context.Background())

		chanOne := make(chan any)
		chanTwo := make(chan any)

		channelOps.MergeOrToOneIgnoreDuplicates(chanOne, chanTwo)

		go func() {
			chanOne <- nil
		}()

		g.Eventually(reader).Should(Receive(BeNil()))
		g.Eventually(reader).Should(BeClosed())
		g.Expect(channelOps.Done()).To(BeClosed())
	})

	t.Run("it cloeses the merged channel if the context is closed", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		channelOps, reader := NewMergeRead[any](ctx)

		chanOne := make(chan any)
		chanTwo := make(chan any)

		channelOps.MergeOrToOneIgnoreDuplicates(chanOne, chanTwo)

		go func() {
			cancel()
		}()

		g.Eventually(reader).ShouldNot(Receive())
		g.Eventually(reader).Should(BeClosed())
		g.Expect(channelOps.Done()).To(BeClosed())
	})

	t.Run("it works with a large number of processes in parallel", func(t *testing.T) {
		done := make(chan struct{})
		wg := new(sync.WaitGroup)
		counter := 10_000
		channelOps, reader := NewMergeRead[any](context.Background())

		channelOps.MergeOrToOneIgnoreDuplicates(nil)

		for i := 0; i < counter; i++ {
			wg.Add(1)
			go func(ii int) {
				defer wg.Done()
				channel := make(chan any)
				_ = channelOps.MergeOrToOne(channel)

				go func(num int, numChan chan any) {
					select {
					case numChan <- num:
					case <-done:
					}
				}(ii, channel)
			}(i)
		}

		wg.Wait()

		g.Eventually(reader).Should(Receive(And(BeNumerically(">=", 0), BeNumerically("<", counter))))
		g.Eventually(reader).Should(BeClosed())
	})

	t.Run("it only adds a channel once, no matter how many times it was attempted to be added", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		channelOpsBeforeCast, reader := NewMergeRead[any](ctx)
		channelOps := channelOpsBeforeCast.(*mergeReadChannelOps[any])

		chanOne := make(chan any)

		channelOps.MergeOrToOneIgnoreDuplicates(chanOne)
		g.Expect(len(channelOps.selectCases)).To(Equal(3))

		g.Expect(channelOps.MergeOrToOneIgnoreDuplicates(chanOne, chanOne)).ToNot(HaveOccurred())
		g.Expect(channelOps.MergeOrToOneIgnoreDuplicates(chanOne)).ToNot(HaveOccurred())
		g.Expect(channelOps.MergeOrToOneIgnoreDuplicates(chanOne)).ToNot(HaveOccurred())

		g.Expect(len(channelOps.selectCases)).To(Equal(3))

		go func() {
			cancel()
		}()

		g.Eventually(reader).ShouldNot(Receive())
		g.Eventually(reader).Should(BeClosed())
	})

	t.Run("it returns an error if the channel has processed or a cancel contet was closed", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		channelOps, reader := NewMergeRead[any](ctx)
		g.Eventually(reader).Should(BeClosed())

		chanOne := make(chan any)
		err := channelOps.MergeOrToOneIgnoreDuplicates(chanOne)
		g.Expect(err).ToNot(BeNil())
		g.Expect(err.Error()).To(Equal("channel has already processed a read operation"))
	})
}
