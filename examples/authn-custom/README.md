## apiserver authentication custom

#### server
```
$ go run ./apiserver-authentication-custom.go
```

#### client
```
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
