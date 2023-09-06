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
		channelOps := NewChannelOps(context.Background())

		chanOne := make(chan any)
		chanTwo := make(chan any)

		go func() {
			chanOne <- "one"
		}()

		go func() {
			chanTwo <- "two"
		}()

		// ensure that whatever value we read in the merge, we can obtain the other value.
		reader := channelOps.MergeOrToOne(chanOne, chanTwo)
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
		channelOps := NewChannelOps(context.Background())

		chanOne := make(chan any)
		chanTwo := make(chan any)

		reader := channelOps.MergeOrToOne(chanOne, chanTwo)

		close(chanOne)
		go func() {
			chanTwo <- "two"
		}()

		g.Eventually(reader).Should(Receive(Equal("two")))
	})

	t.Run("it properly exits if a channel recieved nil", func(t *testing.T) {
		channelOps := NewChannelOps(context.Background())

		chanOne := make(chan any)
		chanTwo := make(chan any)

		reader := channelOps.MergeOrToOne(chanOne, chanTwo)

		go func() {
			chanOne <- nil
		}()

		g.Eventually(reader).Should(Receive(BeNil()))
		g.Eventually(reader).Should(BeClosed())
	})

	t.Run("it cloeses the merged channel if the context is closed", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		channelOps := NewChannelOps(ctx)

		chanOne := make(chan any)
		chanTwo := make(chan any)

		reader := channelOps.MergeOrToOne(chanOne, chanTwo)

		go func() {
			cancel()
		}()

		g.Eventually(reader).ShouldNot(Receive())
		g.Eventually(reader).Should(BeClosed())
	})

	t.Run("it works with a large number of processes in parallel", func(t *testing.T) {
		done := make(chan struct{})
		wg := new(sync.WaitGroup)
		counter := 10_000
		channelOps := NewChannelOps(context.Background())

		reader := channelOps.MergeOrToOne(nil)

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
}
