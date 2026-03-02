package ipc


type Packet struct {
	ID      string         `json:"id"`
	Topic   string         `json:"topic"`
	SentAt  string         `json:"sentAt"`
	Payload map[string]any `json:"payload"`
}


type Frame struct {
	Raw    string
	Bytes  int
	Packet *Packet 
	Err    error   
}
