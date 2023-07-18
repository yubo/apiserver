all: run

.PHONY: run
run:
	go run ./main.go -f ./config.yaml


.PHONY: test
test:
	curl -X GET http://localhost:8080/static/hw

.PHONY: test2
test2:
	curl -X POST http://localhost:8080/api/users -d '{"name":"tom", "age": 16}'
	
