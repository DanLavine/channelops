package channelops

type MergeReadChannelOps[T any] interface {
	// Done returns a channel thats closed when the merged channel reader has processed a single item
	Done() <-chan struct{}

	// Merge any number of read channels into 1 for a single write operation
	//
	// An error is returned if the MergeWriteChannelOps already process a read operation
	MergeOrToOne(orChans ...<-chan T) error

	// Merge any number of read channels into 1 for a single write operation. If the write channel
	// was previously added, this will ignore that channel
	//
	//	// An error is returned if the MergeWriteChannelOps already process a read operation
	MergeOrToOneIgnoreDuplicates(orChans ...<-chan T) error
}
