## apiserver authentication - basic

### server

```sh
$ go run ./main.go -f ./config.yaml
```

### client

#### curl

```sh
$ curl -Ss -u steve:123 http://localhost:8080/hello
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
I0620 13:00:46.333446   72844 main.go:41] "webhook" resp={Name:steve UID: Groups:[dev system:authenticated] Extra:map[]}
```
