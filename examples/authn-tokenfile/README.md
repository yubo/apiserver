## Authentication - token file

### server
```sh
$ go run ./main.go -f ./config.yaml
```


### client

#### curl 

```sh
$ curl -H 'Content-Type:application/json' -H 'Authorization: bearer 123' http://localhost:8080/hello
{"Name":"user3","UID":"uid3","Groups":["group1","group2","system:authenticated"],"Extra":null}
```

#### webhook

```sh
$ cd ./client
$ go run ./main.go --conf ./client.conf
I0522 00:39:16.062719   59275 main.go:41] "webhook" resp={Name:tom@example.com UID:tom@example.com Groups:[developers admins system:authenticated] Extra:map[acme.com/project:[some-project] scope:[openid profile]]}
```
