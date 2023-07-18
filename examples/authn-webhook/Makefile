all: run

.PHONY: run
run:
	go run ./main.go -f ./config.yaml

.PHONY: authn
authn:
	go run ./authn-server/main.go -p 8081

.PHONY: test
test:
	curl -H 'Content-Type:application/json' -H 'Authorization: bearer 123' http://localhost:8080/hello
