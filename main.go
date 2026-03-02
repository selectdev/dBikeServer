package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"dbikeserver/ble"
	"dbikeserver/config"
	"dbikeserver/db"
	"dbikeserver/gpio"
	"dbikeserver/ipc"
	"dbikeserver/script"
	"dbikeserver/util"
)

func cancelOnSignal(cancel context.CancelFunc) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		util.Log("shutting down")
		cancel()
	}()
}

var flagDB = flag.String("db", config.DBPath, "path to the BadgerDB data directory (env: DBIKE_DB_PATH)")

func main() {
	runtime.LockOSThread()
	flag.Parse()
	bleMain()
}

func bleMain() {
	util.Log("dBike Go BLE IPC peripheral booting")
	util.Logf("service=%s write=%s notify=%s", config.ServiceUUID, config.WriteCharUUID, config.NotifyCharUUID)

	database, err := db.Open(*flagDB)
	if err != nil {
		fmt.Fprintln(os.Stderr, "db:", err)
		os.Exit(1)
	}
	defer database.Close()
	util.Logf("db: opened at %s", *flagDB)

	gp, err := gpio.Open()
	if err != nil {
		util.Logf("gpio: not available: %v", err)
		gp = nil
	} else {
		defer gp.Close()
		util.Log("gpio: ready")
	}

	nc := ble.NewNotifyCharacteristic()

	eng, err := script.NewEngine(nc, database, gp, config.ScriptsDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "script engine:", err)
		os.Exit(1)
	}

	wc := ble.NewWriteCharacteristic(func(f ipc.Frame) { handleFrame(nc, eng, f) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cancelOnSignal(cancel)

	if err := ble.RunBLEManager(ctx, nc, wc); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
