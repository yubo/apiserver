all: run

.PHONY: run
run:
	go run ./main.go -f ./config.yaml

.PHONY: server
server:
	go run ./authz-server/main.go

.PHONY: test
test:
	curl  http://localhost:8080/api/v1/users -H "Authorization: Bearer token-admin"

