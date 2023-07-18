all: run

.PHONY: run
run:
	go run ./main.go -f ./config.yaml

.PHONY: server
server:
	go run ./backend-server/main.go

.PHONY: test
test:
	curl -X POST http://localhost:8080/audit/hello -H 'Content-Type: application/json'
