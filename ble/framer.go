package ble

import (
	"bytes"
	"sync"

	"dbikeserver/config"
	"dbikeserver/util"
)

// LineFramer accumulates incoming BLE chunks and splits on newlines,
// mirroring the TypeScript LineFramer. The internal buffer is protected
// by a mutex so concurrent write calls are safe.
type LineFramer struct {
	mu  sync.Mutex
	buf []byte
}

func NewLineFramer() *LineFramer {
	return &LineFramer{}
}

// Append adds chunk to the internal buffer and returns any complete frames
// (newline-terminated, with the newline stripped). If the buffer grows
// beyond MaxFrameBufferBytes without a newline it is reset and nil is returned.
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
