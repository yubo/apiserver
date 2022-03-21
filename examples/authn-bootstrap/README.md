## Authentication - bootstrap

#### server

```sh
$ go run ./authn-bootstrap.go --db-driver=sqlite3 --db-dsn="file:test.db?cache=shared&mode=memory"
```


#### client
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

