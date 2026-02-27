all:
	go build -v
fmt:
	go fmt ./...
startserver:
	./dbikeserver
debugserver:
	go build -o dbikeserver . && ./dbikeserver -debug