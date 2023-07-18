# tracing http

![](./jaeger-snapshot.jpeg)

## server

```sh
$ go run ./main.go  -f ./config.yaml
```

## client

- curl

```sh
$ curl -i http://localhost:8080/api/v3/users/tom
HTTP/1.1 200 OK
Cache-Control: no-cache, private
Content-Type: application/json
Trace-Id: 6c5179733f270984536a98ec347997b4
Date: Tue, 22 Mar 2022 12:21:12 GMT
Content-Length: 18

{
 "Name": "tom"
}
```

## client with otel-col

```sh
$ go run ./client/main.go
2022/03/23 12:58:46 tracer.Start traceID: 0ac9b29f947be6f506f00fd807e5db08
2022/03/23 12:58:46 response traceID: 0ac9b29f947be6f506f00fd807e5db08
```



## install jaeger & otel-col

- https://github.com/yubo/quick-start/tree/main/05-opentelemetry/01-otel-jaeger-promethues

```sh
$ git clone https://github.com/yubo/quick-start.git
$ cd quick-start/05-opentelemetry/01-otel-jaeger-promethues
$ docker-compose up -d
```

