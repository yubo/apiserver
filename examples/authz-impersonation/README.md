## apiserver authentication - impersonation

### server

```sh
$ go run ./main.go -f ./config.yaml
```

### client

#### curl

```sh
$ curl -Ss -u steve:123 \
    -H "Impersonate-User: tom@example.com" \
	-H "Impersonate-Uid: 06f6ce97-e2c5-4ab8-7ba5-7654dd08d52b" \
    -H "Impersonate-Group: developers" \
    -H "Impersonate-Group: admins" \
    -H "Impersonate-Extra-dn: cn=tom,ou=engineers,dc=example,dc=com" \
    -H "Impersonate-Extra-acme.com%2Fproject: some-project" \
    -H "Impersonate-Extra-Scopes: openid" \
    -H "Impersonate-Extra-Scopes: profile" \
    http://localhost:8080/hello
{"Name":"tom@example.com","UID":"06f6ce97-e2c5-4ab8-7ba5-7654dd08d52b","Groups":["developers","admins","system:authenticated"],"Extra":{"acme.com/project":["some-project"],"dn":["cn=tom,ou=engineers,dc=example,dc=com"],"scopes":["openid","profile"]}}
```

#### webhook

```sh
go run ./client/main.go --conf ./client/client.conf
I0620 13:00:46.333446   72844 main.go:41] "webhook" resp={Name:steve UID: Groups:[dev system:authenticated] Extra:map[]}
```


### Reference
- https://kubernetes.io/docs/reference/access-authn-authz/authentication/#user-impersonation
