package channelops

type MergeWriteChannelOps interface {
	// Merge any number of read channels into 1 for a single write operation
	//
	// An error is returned if the MergeWriteChannelOps already process a read operation
	MergeOrToOne(orChans ...<-chan any) error

	// Merge any number of read channels into 1 for a single write operation. If the write channel
	// was previously added, this will ignore that channel
	//
	//	// An error is returned if the MergeWriteChannelOps already process a read operation
	MergeOrToOneIgnoreDuplicates(orChans ...<-chan any) error
}
