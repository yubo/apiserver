## Authentication - webhook

```
   +--------+       +--------+                       +--------------+
   | client | ----> | server |------ webhook ------->| authn-server |
   +--------+       +--------+                       +--------------+
```

### Server

```sh
$ go run ./main.go -f ./config.yaml
```

### Authn Server

```sh
$ go run ./authn-server/main.go -p 8081
Listening 8081 ...

POST /authorize?timeout=30s HTTP/1.1
Host: 127.0.0.1:8081
Accept: application/json, */*
Accept-Encoding: gzip
Authorization: Bearer foobar.circumnavigation
Content-Length: 128
Content-Type: application/json
User-Agent: Go-http-client/1.1

{"metadata":{"creationTimestamp":"0001-01-01T00:00:00Z"},"spec":{"token":"token-777","audiences":["api"]},"status":{"user":{}}}
```

### Client

```sh
$ curl -H 'Content-Type:application/json' -H 'Authorization: bearer token-777' http://localhot:8080/hello
{
  "Name": "test",
  "UID": "u110",
  "Groups": [
    "test:webhook",
    "system:authenticated"
  ],
  "Extra": null
}
```


## See Also
- https://kubernetes.io/zh/docs/reference/access-authn-authz/webhook/
