.PHONY: server client clean
TARGETS=server.out client.out

all: $(TARGETS)

GOFILES=$(shell find .. -name "*.go" -type f)

server.out: $(GOFILES)
	go build -o $@ ./server

client.out: $(GOFILES)
	go build -o $@ ./client

server:
	go run ./server/main.go -logtostderr -v 10

client:
	go run ./client/main.go bash

clean:
	rm -f $(TARGETS)
