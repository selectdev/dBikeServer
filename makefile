UNAME := $(shell uname)

.PHONY: all server panel launcher fmt clean \
        panel-rpi panel-rpi32 \
        server-rpi server-rpi32 \
        startserver startpanel

# Default: build everything.
all: server panel launcher

server:
	go build -v -o dbikeserver .

panel:
	go build -v -o dbikeserver-panel ./panel/

# Raspberry Pi 4/5 and other 64-bit ARM Linux boards.
panel-rpi:
	GOOS=linux GOARCH=arm64 go build -v -o dbikeserver-panel ./panel/

# Raspberry Pi 2/3 running a 32-bit OS.
panel-rpi32:
	GOOS=linux GOARCH=arm GOARM=7 go build -v -o dbikeserver-panel ./panel/

# Server cross-compile targets for RPi.
server-rpi:
	GOOS=linux GOARCH=arm64 go build -v -o dbikeserver .

server-rpi32:
	GOOS=linux GOARCH=arm GOARM=7 go build -v -o dbikeserver .

launcher:
ifeq ($(UNAME), Darwin)
	go build -v -o dbikeserver-launcher ./launcher/
else
	@echo "Skipping dbikeserver-launcher (macOS only)"
endif

fmt:
	go fmt ./...

clean:
	rm -f dbikeserver dbikeserver-panel dbikeserver-launcher

startserver:
	./dbikeserver

startpanel:
	./dbikeserver-panel --watch
