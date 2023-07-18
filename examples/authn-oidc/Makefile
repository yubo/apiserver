all: run

.PHONY: run
run:
	go run ./main.go -f ./config.yaml

.PHONY: op
op:
	make -C ./openid-provider
