.PHONY: run gen-py gen-go client

run:
	go run server/main.go

py-cli: testdata/apidocs.json
	docker run --rm -v `pwd`:/local \
		openapitools/openapi-generator-cli generate \
		-i http://host.docker.internal:8080/apidocs.json \
		-g python \
		-o local/out/python

go-cli: testdata/apidocs.json
	docker run --rm -v `pwd`:/local \
		openapitools/openapi-generator-cli generate \
		-i http://host.docker.internal:8080/apidocs.json \
		-g go \
		-o local/out/go
