package debug

/*
#cgo LDFLAGS: -framework Cocoa
#include <stdlib.h>
#include "debug_window.h"
*/
import "C"
import (
	"encoding/json"
	"unsafe"

	"dbikeserver/ble"
	"dbikeserver/util"
)

// Active enables debug window output. Set to true before calling Run.
var Active bool

// NotifyChar is set in bleMain so GoSendNotify can reach it from ObjC callbacks.
var NotifyChar *ble.NotifyCharacteristic

func init() {
	util.DebugWriter = Write
}

func Run() {
	C.DebugWindowRun()
}

func Stop() {
	if Active {
		C.DebugWindowStop()
	}
}

func Write(line string) {
	if !Active {
		return
	}
	cs := C.CString(line)
	defer C.free(unsafe.Pointer(cs))
	C.DebugWindowAppendLine(cs)
}

// GoSendNotify is called from Objective-C button actions. It decodes the JSON
// payload string and enqueues a BLE notification on the notify characteristic.
//
//export GoSendNotify
func GoSendNotify(ctopic *C.char, cpayload *C.char) {
	if NotifyChar == nil {
		return
	}
	topic := C.GoString(ctopic)
	payloadStr := C.GoString(cpayload)

	var payload map[string]any
	if err := json.Unmarshal([]byte(payloadStr), &payload); err != nil {
		payload = map[string]any{"raw": payloadStr}
	}
	NotifyChar.Notify(topic, payload)
}
