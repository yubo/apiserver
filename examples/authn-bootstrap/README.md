## Authentication - bootstrap(TODO)

#### token file

[tokens.cvs](./tokens.cvs)

```cvs
token-777,user3,uid3,"group1,group2"
```

#### server

```sh
$ go run ./apiserver-authn-tokenfile.go --token-auth-file=./tokens.cvs
```


#### client
```sh
$ curl -H 'Content-Type:application/json' -H 'Authorization: bearer token-777' http://localhost:8080/hello
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

