## restful example

The models storage supported automigrate db.table with resources object

#### server

```sh
go run ./main.go \
	--db-driver=sqlite3 \
	--db-dsn="file:test.db?cache=shared&mode=memory"
```


#### client

```shell
# create
$ curl -X POST -H 'Content-Type:application/json' http://localhost:8080/api/users -d '{"name":"tom", "age": 16}'
{
 "Name": "tom",
 "Age": 16,
 "CreatedAt": "2022-01-09T13:11:29.472816984+08:00",
 "UpdatedAt": "2022-01-09T13:11:29.472817364+08:00"
}

# get
$ curl -X GET http://localhost:8080/api/users/tom
{
 "Name": "tom",
 "Age": 16,
 "CreatedAt": "2022-01-09T13:11:29.472816984+08:00",
 "UpdatedAt": "2022-01-09T13:11:29.472817364+08:00"
}

# list
$ curl -X GET http://localhost:8080/api/users
{
 "total": 1,
 "list": [
  {
   "Name": "tom",
   "Age": 16,
   "CreatedAt": "2022-01-09T13:11:29.472816984+08:00",
   "UpdatedAt": "2022-01-09T13:11:29.472817364+08:00"
  }
 ]
}

# update
$ curl -X PUT -H 'Content-Type:application/json' http://localhost:8080/api/users/tom -d '{"age": 17}'
{
 "Name": "tom",
 "Age": 17,
 "CreatedAt": "2022-01-09T13:11:29.472816984+08:00",
 "UpdatedAt": "2022-01-09T13:12:35.642067489+08:00"
}

# delete
$ curl -X DELETE http://localhost:8080/api/users/tom
{
 "Name": "tom",
 "Age": 17,
 "CreatedAt": "2022-01-09T13:13:04.960265515+08:00",
 "UpdatedAt": "2022-01-09T13:13:11.12705333+08:00"
}

$ curl -X GET http://localhost:8080/api/users
{
 "total": 0,
 "list": null
}
```
