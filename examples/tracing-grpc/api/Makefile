all: api.pb.go  api_grpc.pb.go

%.pb.go: %.proto
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		$<

.PHONY: clean
clean:
	rm -f *.pb.go
