package channelops

type MergeReadChannelOperator[T any] interface {
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

type RepeatableMergeReadChannelOperator[T any] interface {
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
