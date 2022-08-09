# apiserver authz webhook

```
   +--------+       +--------+                       +--------------+
   | client | ----> | server |------ webhook ------->| authz-server |
   +--------+       +--------+                       +--------------+
```

### Server

```sh
$ go run ./main.go -f ./config.yaml
```

### Authz Server

```sh
$ go run ./authz-server/main.go -p 8081
POST /authorize?timeout=30s HTTP/1.1
Host: 127.0.0.1:8081
Accept: application/json, */*
Accept-Encoding: gzip
Authorization: Bearer foobar.circumnavigation
Content-Length: 235
Content-Type: application/json
User-Agent: Go-http-client/1.1

{"metadata":{"creationTimestamp":"0001-01-01T00:00:00Z"},"spec":{"resourceAttributes":{"verb":"list","resource":"users"},"user":"admin","groups":["apiserver:admin","system:authenticated"],"uid":"uid-admin"},"status":{"allowed":false}}
```

### Client

```sh
$ curl  http://localhost:8080/api/v1/users -H "Authorization: Bearer token-admin"
OK
```


## See Also
- https://kubernetes.io/zh/docs/reference/access-authn-authz/webhook/
