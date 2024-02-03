ChannelOPS
----------
[godoc](https://pkg.go.dev/github.com/DanLavine/channelops)

A collection of various channel helper function to incoperate various merge stratagies

#### MergeRead

MergeRead is used when wanting to merge any number of possible read channels into 1 read operation.
This is particularly useful when the read channels are created asyncronously and need to be merged
into the 1 read operation as they become available.