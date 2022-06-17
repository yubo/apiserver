## apiserver authentication custom

### server

```sh
$ go run ./main.go
```

### client

#### curl

```sh
$ curl -Ss  -H 'Authorization: bearer 123' http://localhost:8080/hello
{
 "Name": "system",
 "UID": "",
 "Groups": [
  "system:authenticated"
 ],
 "Extra": null
}
```

#### webhook

```sh
go run ./client/main.go --conf ./client/client.conf
I0617 13:28:35.073152   96903 main.go:41] "webhook" resp={Name:system UID: Groups:[system:authenticated] Extra:map[]}
```
