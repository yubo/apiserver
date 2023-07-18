all: run

.PHONY: run
run:
	go run ./main.go -f ./config.yaml

.PHONY: test
test:
	curl -X POST http://localhost:8080/api/v1/namespaces/test/users -H "Authorization: Bearer token-admin"
	curl -X GET http://localhost:8080/api/v1/namespaces/test/users -H "Authorization: Bearer token-reporter"
	curl -X GET http://localhost:8080/api/v1/namespaces/test/metrics

.PHONY: test2
test2:
	curl -X GET http://localhost:8080/api/v1/namespaces/test/users -H "Authorization: Bearer token-guest"
	curl -X POST http://localhost:8080/api/v1/namespaces/test/users -H "Authorization: Bearer token-reporter"

