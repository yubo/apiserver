all: test.pb.go

%.pb.go: %.proto
	protoc --gofast_out=. --gofast_opt=paths=source_relative \
		$<

.PHONY: clean
clean:
	rm -f *.pb.go

.PHONY: install
install:
	go install github.com/gogo/protobuf/protoc-gen-gofast
