## apiserver authentication - proxy

### server

```sh
$ go run ./main.go -f ./config.yaml
```

### client

#### curl

```sh
$ curl -Ss -X GET \
--key ./testdata/client.key \
--cert ./testdata/client.crt \
--cacert ./testdata/ca.crt \
--header 'Content-Type: application/json' \
--header 'X-Remote-User: tom' \
--header 'X-Remote-Group: dev' \
--header 'X-Remote-Group: admin' \
--header 'X-Remote-Extra-Acme.com%2Fproject: some-project' \
--header 'X-Remote-Extra-Scopes: openid' \
--header 'X-Remote-Extra-Scopes: profile' \
https://127.0.0.1:8443/hello

{"Name":"tom","UID":"","Groups":["dev","admin","system:authenticated"],"Extra":{"acme.com/project":["some-project"],"scopes":["openid","profile"]}}
```

#### webhook

```sh
cd client
go run ./main.go --conf ./client.conf
I0611 03:11:00.893310    3330 main.go:46] "webhook" output={Name:tom UID: Groups:[dev admin system:authenticated] Extra:map[acme.com/project:[some-project] scopes:[openid profile]]}
```
