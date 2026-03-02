package ble

import (
	"encoding/json"
	"time"

	"github.com/go-ble/ble"
	"github.com/google/uuid"

	"dbikeserver/config"
	"dbikeserver/ipc"
	"dbikeserver/util"
)








type NotifyCharacteristic struct {
	sendCh chan []byte
}

func NewNotifyCharacteristic() *NotifyCharacteristic {
	return &NotifyCharacteristic{
		sendCh: make(chan []byte, config.NotifyChannelBuffer),
	}
}


func (nc *NotifyCharacteristic) Handler() ble.NotifyHandlerFunc {
	return func(_ ble.Request, n ble.Notifier) {
		
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
				
				time.Sleep(config.NotifyWriteInterval)
			case <-n.Context().Done():
				util.Log("notify unsubscribed")
				return
			}
		}
	}
}



func (nc *NotifyCharacteristic) Notify(topic string, payload map[string]any) {
	select {
	case nc.sendCh <- encodePacket(topic, payload):
	default:
		util.Logf("notify dropped: %s (buffer full)", topic)
	}
}



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
