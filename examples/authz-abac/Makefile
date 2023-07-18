all: run

.PHONY: run
run:
	go run ./main.go -f ./config.yaml

.PHONY: test
test:
	curl -i -XGET -H 'Authorization: bearer token-777' http://localhost:8080/hello/ro

.PHONY: test2
test2:
	curl -i -XGET http://localhost:8080/hello/ro

