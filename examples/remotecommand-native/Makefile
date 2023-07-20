all: run

.PHONY: run
run:
	go run ./main.go -v 3

.PHONY: test
test:
	go run client/main.go sh

.PHONY: test2
test2:
	go run ./client-attach/main.go $(id)
