all: test

test:
	go vet ./...
