## Audit - webhook

```
   +--------+       +--------+                       +----------------+
   | client | ----> | server |------ webhook ------->| backend-server |
   +--------+       +--------+                       +----------------+
```


server

```sh
$ go run ./main.go -f ./config.yaml
```

backend server
```
$ go run ./backend-server/main.go
Listening 8081 ...

POST /audit/webhook?timeout=30s HTTP/1.1
Host: 127.0.0.1:8081
Accept: application/json, */*
Accept-Encoding: gzip
Authorization: Bearer token.1234567890
Content-Length: 814
Content-Type: application/json
User-Agent: Go-http-client/1.1

{"metadata":{},"items":[{"level":"RequestResponse","auditID":"0df269d3-69c5-4ca5-bed7-8207c2547e27","stage":"RequestReceived","requestURI":"/audit/hello","verb":"post","user":{},"sourceIPs":["127.0.0.1"],"userAgent":"curl/7.79.1","requestReceivedTimestamp":"2022-08-10T14:11:36.208598Z","stageTimestamp":"2022-08-10T14:11:36.208598Z"},{"level":"RequestResponse","auditID":"0df269d3-69c5-4ca5-bed7-8207c2547e27","stage":"ResponseComplete","requestURI":"/audit/hello","verb":"post","user":{},"sourceIPs":["127.0.0.1"],"userAgent":"curl/7.79.1","responseStatus":{"metadata":{},"code":200},"responseObject":"hello, world","requestReceivedTimestamp":"2022-08-10T14:11:36.208598Z","stageTimestamp":"2022-08-10T14:11:36.208848Z","annotations":{"authorization.k8s.io/decision":"allow","authorization.k8s.io/reason":""}}]}
```


client

```sh
$ curl -X POST http://localhost:8081/audit/hello
"hello, world"
```
