## Authentication - bootstrap

### server

```sh
$ go run ./main.go --db-driver=sqlite3 --db-dsn="file:test.db?cache=shared&mode=memory"
```


### client

#### curl

```sh
$ curl -Ss  -H 'Authorization: bearer foobar.circumnavigation' http://localhost:8080/hello
{
 "Name": "system:bootstrap:foobar",
 "UID": "",
 "Groups": [
  "system:bootstrappers",
  "system:bootstrappers:foo",
  "system:authenticated"
 ],
 "Extra": null
}
```

#### webhook

```sh
go run ./client/main.go --conf ./client/client.conf
I0617 13:20:54.310345   94891 main.go:41] "webhook" resp={Name:system:bootstrap:foobar UID: Groups:[system:bootstrappers system:bootstrappers:foo system:authenticated] Extra:map[]}
```
