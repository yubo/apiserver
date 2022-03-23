# OTEL traces

![](./jaeger-snapshot.jpeg)

## server
- config .yaml
```
traces:
  serviceName: otel-traces.examples.apiserver
  contextHeadername: Trace-Id
  jaeger:
    endpoint: http://localhost:14268/api/traces
    insecure: true
```

```
$ go run ./server/apiserver-traces-server.go  -f server/config.yaml
```

## client

- curl
```
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

- client
```
$ go run ./client/apiserver-traces-client.go
2022/03/23 12:58:46 tracer.Start traceID: 0ac9b29f947be6f506f00fd807e5db08
2022/03/23 12:58:46 response traceID: 0ac9b29f947be6f506f00fd807e5db08
```
