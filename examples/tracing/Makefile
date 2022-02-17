DEP_OBJS= ./api/types.pb.go

all: ./api/types.pb.go

.PHONY: run
run: $(DEP_OBJS)
	go run *.go -f ./config.yaml

%.pb.go: %.proto
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		$<
