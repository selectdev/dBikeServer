package util

import (
	"fmt"
	"time"
)

// DebugWriter, if non-nil, receives every log line in addition to stdout.
// The debug package sets this via its init() function.
var DebugWriter func(string)

func Log(msg string) {
	line := fmt.Sprintf("[%s] %s", time.Now().UTC().Format(time.RFC3339), msg)
	fmt.Println(line)
	if DebugWriter != nil {
		DebugWriter(line)
	}
}

func Logf(format string, args ...any) {
	Log(fmt.Sprintf(format, args...))
}
