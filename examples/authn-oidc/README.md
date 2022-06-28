## Apiserver Authentication - OIDC

```
End User(EU) -> Relying Party(RP) -> OpenID Provider(OP)
```

### OIDC - OpenID Provider(OP)

启动 provider，会输出一个用于测试的 Bearer token

```sh
$ cd ./openid-provider
$ go run ./main.go -f ./config.yaml
I0628 19:36:12.950770   43317 main.go:86] "test data" token="eyJhbGciOiJSUzI1NiIsImtpZCI6ImQ5NmNmNThmY2Q5YzZhMmRiYTY1ZjcxZGY4YjhhNjVjZDllM2JlODEyNzY5NTE4NGZlNjI2OWI4OWZjYzQzZDAifQ.ewogICJpc3MiOiAiaHR0cDovL2xvY2FsaG9zdDo4MDgxIiwKICAiYXVkIjogIm15LWNsaWVudCIsCiAgInVzZXJuYW1lIjogInN0ZXZlIiwKICAiZ3JvdXBzIjogWyJ0ZWFtMSIsICJ0ZWFtMiJdLAogICJleHAiOiAxNjU2NTAyNTcyCn0K.PJ3ZYlycyp3cBfphGxhfs1nKGoAOHCusK9KXFxVi8galRggm_3v7JJudYEL2nAjhp7SyHJiBm272FWOgt_bW-XGpRZXBw1uTuYLLlLS9oo2qff54MVdbzchS0zIEp6_fi7jEXZvUXW0VJ8tXQwvAud-hny80dwsYXq7ei1j3_Z9aoYMagLXuMka4dPDrpRCp6Ue-KDcjeBtPtNrW_jWcTvlBRSnCEUYfdW1rdwP5wvC9kQz6NmjqBkdJYjUUwnrqe4Sy2E9M7dIPJqSVD1dBh76tz9r7HkDPFc0BsCkYAhqFVuOyHAlq51vX5Fk9ojYTErRYRRWWzXsVNx9tZg4uzg"
I0628 18:46:16.937476   18728 deprecated_insecure_serving.go:52] Serving insecurely on [::]:8081
```

### Server - Relaying Party(RP)

```sh
$ go run ./main.go -f ./config.yaml
```

### Client - End User(EU)

#### curl

使用 bearer token 访问

```sh
$ TOKEN=eyJhbGciOiJSUzI1NiIsImtpZCI6ImQ5NmNmNThmY2Q5YzZhMmRiYTY1ZjcxZGY4YjhhNjVjZDllM2JlODEyNzY5NTE4NGZlNjI2OWI4OWZjYzQzZDAifQ.ewogICJpc3MiOiAiaHR0cDovL2xvY2FsaG9zdDo4MDgxIiwKICAiYXVkIjogIm15LWNsaWVudCIsCiAgInVzZXJuYW1lIjogInN0ZXZlIiwKICAiZ3JvdXBzIjogWyJ0ZWFtMSIsICJ0ZWFtMiJdLAogICJleHAiOiAxNjU2NTAyNTcyCn0K.PJ3ZYlycyp3cBfphGxhfs1nKGoAOHCusK9KXFxVi8galRggm_3v7JJudYEL2nAjhp7SyHJiBm272FWOgt_bW-XGpRZXBw1uTuYLLlLS9oo2qff54MVdbzchS0zIEp6_fi7jEXZvUXW0VJ8tXQwvAud-hny80dwsYXq7ei1j3_Z9aoYMagLXuMka4dPDrpRCp6Ue-KDcjeBtPtNrW_jWcTvlBRSnCEUYfdW1rdwP5wvC9kQz6NmjqBkdJYjUUwnrqe4Sy2E9M7dIPJqSVD1dBh76tz9r7HkDPFc0BsCkYAhqFVuOyHAlq51vX5Fk9ojYTErRYRRWWzXsVNx9tZg4uzg
$ curl -Ss  -H "Authorization: bearer ${TOKEN}" http://localhost:8080/hello
{
 "Name": "http://localhost:8081#steve",
 "UID": "",
 "Groups": [
  "team1",
  "team2",
  "system:authenticated"
 ],
 "Extra": null
}
```

## references
- https://openid.net/specs/openid-connect-core-1_0.html
