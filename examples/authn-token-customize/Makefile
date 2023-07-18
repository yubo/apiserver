all: run

.PHONY: run
run:
	go run ./main.go -f ./config.yaml

.PHONY: test
test:
	curl -Ss  -H 'Authorization: bearer 123' http://localhost:8080/hello

.PHONY: client
client:
	make -C ./client
