package ipc

// Packet is the JSON envelope exchanged over BLE.
type Packet struct {
	ID      string         `json:"id"`
	Topic   string         `json:"topic"`
	SentAt  string         `json:"sentAt"`
	Payload map[string]any `json:"payload"`
}

// Frame is the result of parsing one newline-delimited chunk.
type Frame struct {
	Raw    string
	Bytes  int
	Packet *Packet // non-nil on success
	Err    error   // non-nil on parse failure
}
