all: run

.PHONY: run
run:
	go run ./main.go -f ./config.yaml -v 10

.PHONY: test
test:
	curl -Ss -u steve:123 \
		-H "Impersonate-User: tom@example.com" \
		-H "Impersonate-Uid: 06f6ce97-e2c5-4ab8-7ba5-7654dd08d52b" \
		-H "Impersonate-Group: developers" \
		-H "Impersonate-Group: admins" \
		-H "Impersonate-Extra-dn: cn=tom,ou=engineers,dc=example,dc=com" \
		-H "Impersonate-Extra-acme.com%2Fproject: some-project" \
		-H "Impersonate-Extra-Scopes: openid" \
		-H "Impersonate-Extra-Scopes: profile" \
		http://localhost:8080/hello
