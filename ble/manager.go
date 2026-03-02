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





func RunBLEManager(ctx context.Context, nc *NotifyCharacteristic, wc *WriteCharacteristic) error {
	d, err := darwin.NewDevice()
	if err != nil {
		return fmt.Errorf("new BLE device: %w", err)
	}
	ble.SetDefaultDevice(d)

	
	svc := ble.NewService(ble.MustParse(config.ServiceUUID))

	writeCh := ble.NewCharacteristic(ble.MustParse(config.WriteCharUUID))
	writeCh.HandleWrite(wc.Handler())
	
	
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
			return nil 
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
