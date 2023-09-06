package channelops

import (
	"context"
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

		reader := channelOps.MergeOrToOne(chanOne, chanTwo)
		value := <-reader
		switch value.(string) {
		case "one":
			g.Eventually(chanTwo).Should(Receive(Equal("one")))
		case "two":
			g.Eventually(chanOne).Should(Receive(Equal("one")))
		}
	})
}
