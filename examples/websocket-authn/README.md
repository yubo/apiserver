## Authentication - websocket

#### server

```sh
$ go run ./server/main.go
I0321 20:14:09.742701   67741 server.go:188] external host was not specified, using yubo-didi-mbp16
I0321 20:14:09.743553   67741 openapi.go:123] "route register" method="GET" path="/hello"
I0321 20:14:09.743578   67741 openapi.go:474] add scheme BearerToken
I0321 20:14:09.743678   67741 deprecated_insecure_serving.go:52] Serving insecurely on [::]:8080
I0321 20:14:12.841374   67741 log.go:184] http: response.WriteHeader on hijacked connection from github.com/yubo/apiserver/pkg/server/httplog.(*respLogger).WriteHeader (httplog.go:217)
```


#### client
```sh
$ go run ./client/main.go
I0321 20:14:12.841372   67765 websocket-authn-client.go:64] "recv" contain="username: test, groups: [system:authenticated]"

```

