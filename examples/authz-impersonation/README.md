## apiserver authentication - impersonation

### server

```sh
$ go run ./main.go -f ./config.yaml
```

### client

#### curl

```sh
# login admin as tom@example.com
curl -Ss \
	-H "Authorization: Bearer token-admin" \
	-H "Impersonate-User: tom@example.com" \
	-H "Impersonate-Uid: 06f6ce97-e2c5-4ab8-7ba5-7654dd08d52b" \
	-H "Impersonate-Group: group-reporter" \
	-H "Impersonate-Group: admin" \
	-H "Impersonate-Extra-dn: cn=tom,ou=engineers,dc=example,dc=com" \
	-H "Impersonate-Extra-Scopes: openid" \
	-H "Impersonate-Extra-Scopes: profile" \
	http://localhost:8080/hello

{"Name":"tom@example.com","UID":"06f6ce97-e2c5-4ab8-7ba5-7654dd08d52b","Groups":["group-reporter","admin","system:authenticated"],"Extra":{"dn":["cn=tom,ou=engineers,dc=example,dc=com"],"scopes":["openid","profile"]}}


# login reporter
curl -Ss \
	-H "Authorization: Bearer token-reporter" \
	http://localhost:8080/hello

{"Name":"reporter","UID":"uid-reporter","Groups":["group-reporter","system:authenticated"],"Extra":null}

```

#### webhook

```sh
go run ./client/main.go --conf ./client/client.conf
I0611 15:08:28.220759   39579 main.go:30] "webhook" resp={Name:tom@example.com UID:06f6ce97-e2c5-4ab8-7ba5-7654dd08d52b Groups:[group-reporter admin system:authenticated] Extra:map[scopes:[openid profile]]}
```


### Reference
- https://kubernetes.io/docs/reference/access-authn-authz/authentication/#user-impersonation
