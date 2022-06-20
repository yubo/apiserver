all: run

.PHONY: run
run:
	go run ./main.go --db-driver=sqlite3 --db-dsn="file:test.db?cache=shared&mode=memory"
