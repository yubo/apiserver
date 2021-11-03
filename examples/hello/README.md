## hello world

#### With HTTPS
```sh
# server
$ go run ./apiserver-hello-world.go --cert-dir=/tmp/hello-world

# client
$ curl -k https://localhost:8443/hello
"hello, world"

# client with CA Cert
$ curl --cacert /tmp/hello-world/apiserver.crt https://localhost:8443/hello
```

#### HTTP
```
$ go run ./apiserver-hello-world.go --cert-dir=/tmp/hello-world --address=0.0.0.0 --port=8080
$ curl http://localhost:8080/hello
```
