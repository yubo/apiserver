all: run

.PHONY: run
run:
	go run ./main.go -f ./config.yaml

.PHONY: test
test:
	curl -X POST -H 'Content-Type: application/json' --cacert ./testdata/ca.crt --cert ./testdata/client.crt --key ./testdata/client.key  https://127.0.0.1:8443/inc -d '{"X": 1}'

.PHONY: test2
test2:
	make -C ./client
