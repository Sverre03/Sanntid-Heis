package msgidbuffer

const bufferSize = 5

type MessageIDBuffer struct {
	messageIDs [bufferSize]uint64
	size       int
	index      int
}

func (buf *MessageIDBuffer) Add(id uint64) {
	if buf.size == buf.index {
		buf.index = 0
	}
	buf.messageIDs[buf.index] = id
	buf.index += 1
}

func (buf *MessageIDBuffer) Contains(id uint64) bool {
	for i := 0; i < buf.size; i++ {
		if buf.messageIDs[i] == id {
			return true
		}
	}
	return false
}
