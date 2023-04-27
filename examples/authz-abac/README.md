## APIserver - Authorization - ABAC(Attribute-based access control)

This example shows the minimal code needed to get a restful.WebService working.

## server
```sh
$ go run ./main.go -f ./config.yaml
```


#### client

```sh
$ curl -i -XGET -H 'Authorization: bearer token-777' http://localhost:8080/hello/ro
HTTP/1.1 200 OK
Cache-Control: no-cache, private
Date: Wed, 10 Nov 2021 17:51:35 GMT
Content-Length: 6
Content-Type: text/plain; charset=utf-8

hello
```

```sh
$ curl -XGET  -Ss -i http://localhost:8080/hello/ro
HTTP/1.1 401 Unauthorized
Cache-Control: no-cache, private
Content-Type: application/json
Date: Wed, 10 Nov 2021 17:52:00 GMT
Content-Length: 123

{
  "metadata": {

  },
  "status": "Failure",
  "message": "Unauthorized",
  "reason": "Unauthorized",
  "code": 401
}
```

## References
- https://kubernetes.io/docs/reference/access-authn-authz/abac/

