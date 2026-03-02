package ble

import (
	"bytes"
	"sync"

	"dbikeserver/config"
	"dbikeserver/util"
)




type LineFramer struct {
	mu  sync.Mutex
	buf []byte
}

func NewLineFramer() *LineFramer {
	return &LineFramer{}
}




func (f *LineFramer) Append(chunk []byte) [][]byte {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.buf = append(f.buf, chunk...)

	if len(f.buf) > config.MaxFrameBufferBytes {
		util.Logf("framer buffer exceeded %d bytes without newline; resetting", config.MaxFrameBufferBytes)
		f.buf = nil
		return nil
	}

	var frames [][]byte
	for {
		idx := bytes.IndexByte(f.buf, '\n')
		if idx < 0 {
			break
		}
		frame := make([]byte, idx)
		copy(frame, f.buf[:idx])
		frames = append(frames, frame)
		f.buf = f.buf[idx+1:]
	}
	return frames
}
