# models with db
支持sql的storage

```
go run ./main.go --db-driver=sqlite3 --db-dsn="file:test.db?cache=shared&mode=memory"

I0321 19:11:21.653536   17865 apiserver-models.go:53] "create" name="bootstrap-token-test" type=bootstrap.kubernetes.io/token
I0321 19:11:21.653659   17865 apiserver-models.go:59] "get" name="bootstrap-token-test" type=bootstrap.kubernetes.io/token
I0321 19:11:21.653809   17865 apiserver-models.go:66] "get" name="bootstrap-token-test" type=bootstrap.kubernetes.io/token/2
```
