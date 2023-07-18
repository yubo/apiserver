all: run

.PHONY: run
run:
	go run ./main.go -f ./config.yaml

.PHONY: test
test:
	curl -Ss -X GET \
		--key ./testdata/client.key \
		--cert ./testdata/client.crt \
		--cacert ./testdata/ca.crt \
		--header 'Content-Type: application/json' \
		--header 'X-Remote-User: tom' \
		--header 'X-Remote-Group: dev' \
		--header 'X-Remote-Group: admin' \
		--header 'X-Remote-Extra-Acme.com%2Fproject: some-project' \
		--header 'X-Remote-Extra-Scopes: openid' \
		--header 'X-Remote-Extra-Scopes: profile' \
		https://127.0.0.1:8443/hello


