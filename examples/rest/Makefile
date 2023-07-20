.PHONY: run test test2 test3
run:
	go run ./main.go -f ./config.yaml

create:
	curl -X POST -H 'Content-Type:application/json' http://localhost:8080/api/v1/users -d '{"name":"tom", "age": 16}'


get:
	curl -X GET http://localhost:8080/api/v1/users/tom
	curl -X GET http://localhost:8080/api/v2/users/tom

list:
	curl -X GET http://localhost:8080/api/v1/users -G --data-urlencode 'query=name in (tom, jerry),age>15'

update:
	curl -X PUT -H 'Content-Type:application/json' http://localhost:8080/api/v1/users/tom -d '{"age": 17}'

del:
	curl -X DELETE http://localhost:8080/api/v1/users/tom

