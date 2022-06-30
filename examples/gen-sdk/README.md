# generate sdk

This is an example of generate api-python, api-golang

## server

```sh
go run server/main.go
```

## generate sdk 

#### python

```sh
docker run --rm -v `pwd`:/local \
	openapitools/openapi-generator-cli generate \
	-i http://host.docker.internal:8080/apidocs.json \
	-g python \
	-o local/out/python
```

#### golang

```sh
docker run --rm -v `pwd`:/local \
	openapitools/openapi-generator-cli generate \
	-i http://host.docker.internal:8080/apidocs.json \
	-g go \
	-o local/out/go
```

## client - using sdk

#### golang

```sh
$ go run client/main.go
Response from `UserApi.CreateUser`: {Hamilton Ham 0086-123456}
Response from `UserApi.GetUser`: {Hamilton Ham 0086-123456}
Response from `UserApi.UpdateUser`: {Hamilton Ham 0086-888888}
Response from `UserApi.GetUsers`: {[{Hamilton Ham 0086-888888}] 1}
Response from `UserApi.DeleteUser`: {Hamilton Ham 0086-888888}
```

## See Also
- https://github.com/openapitools/openapi-generator
- https://openapi-generator.tech/docs/generators
