## hello world

#### With HTTPS
```sh
# server
$ go run ./apiserver-hello-world.go --cert-dir=/tmp/hello-world

# client
$ curl --cacert /tmp/hello-world/apiserver.crt https://localhost:8443/hello
hello, world
```

#### HTTP
```sh
# server
$ go run ./apiserver-hello-world.go --secure-serving=false --insecure-serving

# client
$ curl http://localhost:8080/hello
hello, world
```
