package channelops

type MergeReadChannelOps[T any] interface {
	// RETURNS:
	// - <-chan struct{} - channel to indicate if this merger read operation has processed and will no longer process
	//
	// Done can be used to detemerine if the channel has been read from and will no longer process anymore read operations
	Done() <-chan struct{}

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
	MergeOrToOne(orChans ...<-chan T) error

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
	MergeOrToOneIgnoreDuplicates(orChans ...<-chan T) error
}

type RepeatableMergeReadChannelOps[T any] interface {
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
