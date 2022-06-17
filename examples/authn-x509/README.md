## authn-x509-webhook

### generate cert files

详情见 [testdata/gencerts.sh](./testdata/gencerts.sh)

为 user: tom, groups: [dev, admin] 创建证书

```sh
openssl genrsa -out client.key 2048
openssl req -new -key client.key -out client.csr -subj "/CN=tom/O=dev/O=admin" -config client.conf
openssl x509 -req -in client.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out client.crt -days 100000 -extensions v3_req -extfile client.conf
```

### server

```sh
$ go run ./main.go -f ./config.yaml
```


### client

#### curl

```sh
$ curl -X POST -H 'Content-Type:application/json' --cacert ./testdata/ca.crt --cert ./testdata/client.crt --key ./testdata/client.key  https://127.0.0.1:8443/inc -d '{"X": 1}'
{
 "X": 2,
 "User": {
  "Name": "tome",
  "UID": "",
  "Groups": [
   "dev",
   "admin",
   "system:authenticated"
  ],
  "Extra": null
}
```

#### webhook

```sh
$ cd client
$ go run ./main.go
I0617 14:34:54.673198    8181 main.go:51] "webhook" input={X:1} output={X:2 User:{Name:tome UID: Groups:[dev admin system:authenticated] Extra:map[]}}
```

