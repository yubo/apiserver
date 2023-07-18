## custom response writer

#### start server

```sh
$ go run ./main.go
```

#### 


default response writer

```
$ curl -Ss http://localhost:8080/api/v1/users/tom
{"name":"tom","nickName":null,"phone":"12345"}
``````


custom response writer `umi.RespWriter`

```
curl -Ss http://localhost:8080/api/v2/users/tom
{"data":{"name":"tom","nickName":null,"phone":"12345"},"host":"yubo-didi-mbp16","success":true,"traceId":"00000000000000000000000000000000"}
```
