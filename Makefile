all: run

.PHONY: run
run:
	go vet ./...
