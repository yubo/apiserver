## Authentication - token file

### server

[tokens.cvs](./tokens.cvs)

```cvs
token-777,user3,uid3,"group1,group2"
```

```sh
$ go run ./main.go -f ./config.yaml
```


### client

#### curl 

```sh
$ curl -H 'Content-Type:application/json' -H 'Authorization: bearer 123' http://localhost:8080/hello
{
 "Name": "user3",
 "UID": "uid3",
 "Groups": [
  "group1",
  "group2",
  "system:authenticated"
 ],
 "Extra": null
}
```

#### webhook

```sh
$ cd ./client
$ go run ./main.go --conf ./client.conf
I0617 13:31:56.412910   98328 main.go:41] "webhook" resp={Name:user3 UID:uid3 Groups:[group1 group2 system:authenticated] Extra:map[]}
```
