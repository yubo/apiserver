all: regenerate

regenerate:
	go install github.com/gogo/protobuf/protoc-min-version
	protoc-min-version --version="3.0.0" --gogo_out=paths=source_relative:. test.proto
