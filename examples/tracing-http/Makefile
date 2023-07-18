.PHONY: run
run:
	go run ./main.go -f ./config.yaml

.PHONY: test
test:
	curl -i http://localhost:8080/api/v3/users/tom

# must install otel
.PHONY: test2
test2:
	go run ./client/main.go
