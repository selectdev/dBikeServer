UNAME := $(shell uname)

.PHONY: all server panel fmt clean \
        panel-rpi \
        server-rpi \
        startserver startpanel

# Default: build everything.
all: server panel

server:
	go build -v -o dbikeserver .

panel:
	go build -v -o dbikeserver-panel ./panel/

# Raspberry Pi 4/5 and other 64-bit ARM Linux boards.
panel-rpi:
	GOOS=linux GOARCH=arm64 go build -v -o dbikeserver-panel ./panel/

# Server cross-compile targets for RPi.
server-rpi:
	GOOS=linux GOARCH=arm64 go build -v -o dbikeserver .

fmt:
	go fmt ./...

clean:
	rm -f dbikeserver dbikeserver-panel