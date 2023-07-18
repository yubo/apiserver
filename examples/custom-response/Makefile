.PHONY: run
run:
	go run main.go

.PHONY: test
test:
	curl -Ss http://localhost:8080/api/v1/users/tom

.PHONY: test2
test2:
	curl -Ss http://localhost:8080/api/v2/users/tom 

# with trace
.PHONY: test3
test3:
	curl -Ss -H "Traceparent: 00-2fe3c8afacbedd9f2d2e5af9b8aef9e1-2b708c8bbe73fa74-01" http://localhost:8080/api/v2/users/tom 
