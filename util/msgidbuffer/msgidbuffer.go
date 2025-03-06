package msgidbuffer

const bufferSize = 5

// A buffer that holds the last #buffersize message ids
type MessageIDBuffer struct {
	messageIDs [bufferSize]uint64
	size       int
	index      int
}

// using Add, you can add a message ID to the buffer. It overwrites in a FIFO manner
func (buf *MessageIDBuffer) Add(id uint64) {
	if buf.size == buf.index {
		buf.index = 0
	}
	buf.messageIDs[buf.index] = id
	buf.index += 1
}

// check if a message id is in the buffer
func (buf *MessageIDBuffer) Contains(id uint64) bool {
	for i := 0; i < buf.size; i++ {
		if buf.messageIDs[i] == id {
			return true
		}
	}
	return false
}
