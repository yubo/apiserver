## APIserver - Authorization - Path

This example shows the minimal code needed to get a restful.WebService working.

## server
```sh
$ go run ./main.go --authorization-always-allow-paths /hello/ro --authorization-mode AlwaysDeny
```


#### client

```sh
$ curl -i http://localhost:8080/hello/ro
HTTP/1.1 200 OK
Cache-Control: no-cache, private
Date: Wed, 10 Nov 2021 08:45:15 GMT
Content-Length: 6
Content-Type: text/plain; charset=utf-8

hello

$ curl -XGET  -Ss -i http://localhost:8080/hello/deny
HTTP/1.1 500 Internal Server Error
Cache-Control: no-cache, private
Content-Type: text/plain; charset=utf-8
X-Content-Type-Options: nosniff
Date: Wed, 10 Nov 2021 08:46:56 GMT
Content-Length: 58

Internal Server Error: "/hello/deny": no user on request.
```
