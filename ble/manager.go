package ble

import (
	"context"
	"fmt"
	"time"

	"github.com/go-ble/ble"
	"github.com/go-ble/ble/darwin"

	"dbikeserver/config"
	"dbikeserver/util"
)

// RunBLEManager initialises the CoreBluetooth device, registers the GATT
// service, and advertises until ctx is cancelled. If advertising stops
// unexpectedly (e.g. after a client disconnects) it restarts automatically
// after AdvertisingRecoveryDelay, mirroring the Node.js recovery logic.
func RunBLEManager(ctx context.Context, nc *NotifyCharacteristic, wc *WriteCharacteristic) error {
	d, err := darwin.NewDevice()
	if err != nil {
		return fmt.Errorf("new BLE device: %w", err)
	}
	ble.SetDefaultDevice(d)

	// Build GATT service.
	svc := ble.NewService(ble.MustParse(config.ServiceUUID))

	writeCh := ble.NewCharacteristic(ble.MustParse(config.WriteCharUUID))
	writeCh.HandleWrite(wc.Handler())
	// Also advertise write-without-response (CharWriteNR) so the iOS app can
	// use the faster path. The same handler processes both write types.
	writeCh.Property |= ble.CharWriteNR

	notifyCh := ble.NewCharacteristic(ble.MustParse(config.NotifyCharUUID))
	notifyCh.HandleNotify(nc.Handler())

	svc.AddCharacteristic(writeCh)
	svc.AddCharacteristic(notifyCh)

	if err := ble.AddService(svc); err != nil {
		return fmt.Errorf("add service: %w", err)
	}

	serviceUUID := ble.MustParse(config.ServiceUUID)
	reason := "initial"

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		util.Logf("advertising as %q (%s)", config.DeviceName, reason)
		advErr := ble.AdvertiseNameAndServices(ctx, config.DeviceName, serviceUUID)

		if ctx.Err() != nil {
			return nil // graceful shutdown via SIGINT/SIGTERM
		}

		if advErr != nil {
			util.Logf("advertising stopped (%v); recovering in %s", advErr, config.AdvertisingRecoveryDelay)
		} else {
			util.Logf("advertising stopped; recovering in %s", config.AdvertisingRecoveryDelay)
		}

		select {
		case <-ctx.Done():
			return nil
		case <-time.After(config.AdvertisingRecoveryDelay):
			reason = "recovery"
		}
	}
}
