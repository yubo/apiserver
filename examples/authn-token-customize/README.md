## apiserver authentication - token customize

### server

```sh
$ go run ./main.go
```

### client

#### curl

```sh
$ curl -Ss  -H 'Authorization: bearer 123' http://localhost:8080/hello
{
 "Name": "steve",
 "UID": "",
 "Groups": [
  "dev",
  "system:authenticated"
 ],
 "Extra": null
}
```

#### webhook

```sh
go run ./client/main.go --conf ./client/client.conf
I0620 12:24:09.061620   50854 main.go:41] "webhook" resp={Name:steve UID: Groups:[dev system:authenticated] Extra:map[]}
```
