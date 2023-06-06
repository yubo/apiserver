## Authentication - password file

### server
```sh
$ go run ./main.go -f ./config.yaml
```


### client

#### curl 

```sh
$ curl -Ss -u user3:123 http://localhost:8080/hello
{"Name":"user3","UID":"uid3","Groups":["group1","group2","system:authenticated"],"Extra":null}
```

#### webhook

```sh
$ cd ./client
$ go run ./main.go --conf ./client.conf
I0522 01:28:46.350299   64559 main.go:45] "webhook" resp={Name:user3 UID:uid3 Groups:[group1 group2 system:authenticated] Extra:map[]}
```
