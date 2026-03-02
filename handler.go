package main

import (
	"dbikeserver/ble"
	"dbikeserver/ipc"
	"dbikeserver/script"
	"dbikeserver/util"
)

func handleFrame(nc *ble.NotifyCharacteristic, eng *script.Engine, f ipc.Frame) {
	if f.Err != nil {
		util.Logf("failed to decode inbound frame (%d bytes): %v", f.Bytes, f.Err)
		nc.Notify("ipc.error", map[string]any{
			"source":  "go",
			"reason":  "json_parse_failed",
			"rxBytes": f.Bytes,
		})
		return
	}

	topic := f.Packet.Topic
	util.Logf("rx topic=%s bytes=%d", topic, f.Bytes)

	nc.Notify("ack", map[string]any{
		"source":  "go",
		"rxTopic": topic,
		"rxBytes": f.Bytes,
	})

	if eng.HandleEvent(topic, f.Packet.Payload) {
		return
	}

	switch topic {
	case "ping":
		nc.Notify("pong", map[string]any{
			"source":   "go",
			"sequence": f.Packet.Payload["sequence"],
		})
	}
}
