package config

import (
	"os"
	"time"
)

const (
	ServiceUUID    = "7a0f77a7-e0a2-4a3b-bd65-9fa0e3a03001"
	WriteCharUUID  = "7a0f77a7-e0a2-4a3b-bd65-9fa0e3a03002"
	NotifyCharUUID = "7a0f77a7-e0a2-4a3b-bd65-9fa0e3a03003"

	MaxFrameBufferBytes      = 1024 * 1024
	AdvertisingRecoveryDelay = 350 * time.Millisecond

	NotifyChannelBuffer = 32

	NotifyWriteInterval = 20 * time.Millisecond
)

var DeviceName = envOr("DBIKE_BLE_NAME", "dBike-Go")

var ScriptsDir = envOr("DBIKE_SCRIPTS_DIR", "scripts")

var DBPath = envOr("DBIKE_DB_PATH", "./data")

var OpenAIAPIKey = os.Getenv("OPENAI_API_KEY")

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
