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
	dbg "dbikeserver/debug"
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

var (
	flagDebug = flag.Bool("debug", false, "open a native debug console window")
	flagDB    = flag.String("db", "./data", "path to the BadgerDB data directory")
)

func main() {
	// AppKit requires all UI calls to originate from the main OS thread.
	// LockOSThread pins this goroutine to the current OS thread for the
	// lifetime of the process, satisfying that requirement.
	runtime.LockOSThread()

	flag.Parse()

	if *flagDebug {
		dbg.Active = true
		go bleMain()
		dbg.Run()
	} else {
		bleMain()
	}
}

func bleMain() {
	defer dbg.Stop()

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
	dbg.NotifyChar = nc

	eng, err := script.NewEngine(nc, database, gp, "scripts")
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
