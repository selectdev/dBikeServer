package ble

import (
	"encoding/json"
	"time"

	"github.com/go-ble/ble"
	"github.com/google/uuid"

	"dbikeserver/ipc"
	"dbikeserver/util"
)

// NotifyCharacteristic manages the server-to-client notification channel.
//
// All writes to ble.Notifier happen exclusively on the handler goroutine
// (the one go-ble/ble spawns when the central subscribes). External callers
// enqueue packets via the sendCh channel; the handler goroutine drains it.
// This avoids concurrent CGo calls into CoreBluetooth, which caused the
// segfault, and gives us a natural place to retry on "tx queue full".
type NotifyCharacteristic struct {
	sendCh chan []byte
}

func NewNotifyCharacteristic() *NotifyCharacteristic {
	return &NotifyCharacteristic{
		sendCh: make(chan []byte, 32),
	}
}

// notifyWriteInterval is the minimum gap between successive n.Write calls.
// CoreBluetooth's internal TX queue is small; writing two notifications
// back-to-back (e.g. "ack" + "pong") fills it and causes a segfault inside
// cbgo when the second call is attempted. 20 ms is well above the minimum
// BLE connection interval so the queue is always drained in time.
const notifyWriteInterval = 20 * time.Millisecond

// Handler returns the ble.NotifyHandlerFunc to register on the characteristic.
func (nc *NotifyCharacteristic) Handler() ble.NotifyHandlerFunc {
	return func(_ ble.Request, n ble.Notifier) {
		// Drain any stale packets queued during a previous connection.
		for len(nc.sendCh) > 0 {
			<-nc.sendCh
		}

		util.Log("notify subscribed")
		nc.Notify("sim.ready", map[string]any{
			"source":  "go",
			"message": "IPC ready",
		})

		for {
			select {
			case data := <-nc.sendCh:
				if _, err := n.Write(data); err != nil {
					util.Logf("notify write error: %v", err)
				}
				// Pace writes so the BLE TX queue never fills.
				time.Sleep(notifyWriteInterval)
			case <-n.Context().Done():
				util.Log("notify unsubscribed")
				return
			}
		}
	}
}

// Notify enqueues a packet for the subscribed central. Non-blocking: if the
// send buffer is full the packet is dropped with a log line.
func (nc *NotifyCharacteristic) Notify(topic string, payload map[string]any) {
	select {
	case nc.sendCh <- encodePacket(topic, payload):
	default:
		util.Logf("notify dropped: %s (buffer full)", topic)
	}
}

// WriteCharacteristic receives client-to-server writes, assembles newline-
// delimited frames via LineFramer, parses each frame as JSON, and calls onFrame.
type WriteCharacteristic struct {
	onFrame func(ipc.Frame)
	framer  *LineFramer
}

func NewWriteCharacteristic(onFrame func(ipc.Frame)) *WriteCharacteristic {
	return &WriteCharacteristic{
		onFrame: onFrame,
		framer:  NewLineFramer(),
	}
}

// Handler returns the ble.WriteHandlerFunc to register on the characteristic
// (works for both write-with-response and write-without-response).
func (wc *WriteCharacteristic) Handler() ble.WriteHandlerFunc {
	return func(req ble.Request, _ ble.ResponseWriter) {
		for _, raw := range wc.framer.Append(req.Data()) {
			if len(raw) == 0 {
				continue
			}
			var pkt ipc.Packet
			if err := json.Unmarshal(raw, &pkt); err != nil {
				wc.onFrame(ipc.Frame{Raw: string(raw), Bytes: len(raw), Err: err})
			} else {
				wc.onFrame(ipc.Frame{Raw: string(raw), Bytes: len(raw), Packet: &pkt})
			}
		}
	}
}

func encodePacket(topic string, payload map[string]any) []byte {
	if payload == nil {
		payload = map[string]any{}
	}
	pkt := ipc.Packet{
		ID:      uuid.New().String(),
		Topic:   topic,
		SentAt:  time.Now().UTC().Format(time.RFC3339Nano),
		Payload: payload,
	}
	data, _ := json.Marshal(pkt)
	return append(data, '\n')
}
