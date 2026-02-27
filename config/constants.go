package config

import (
	"os"
	"time"
)

const (
	ServiceUUID    = "7a0f77a7-e0a2-4a3b-bd65-9fa0e3a03001"
	WriteCharUUID  = "7a0f77a7-e0a2-4a3b-bd65-9fa0e3a03002"
	NotifyCharUUID = "7a0f77a7-e0a2-4a3b-bd65-9fa0e3a03003"

	MaxFrameBufferBytes      = 1024 * 1024 // 1 MB
	AdvertisingRecoveryDelay = 350 * time.Millisecond
)

var DeviceName = func() string {
	if name := os.Getenv("DBIKE_BLE_NAME"); name != "" {
		return name
	}
	return "dBike-Go"
}()
