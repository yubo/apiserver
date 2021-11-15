# apiserver authz webhook
- https://kubernetes.io/zh/docs/reference/access-authn-authz/webhook/

## Authentication
[tokens.cvs](./tokens.cvs)
```csv
token-admin,admin,uid-admin,"apiserver:admin"
token-reporter,reporter,uid-reporter,"apiserver:reporter"
token-guest,guest,uid-guest,"apiserver:guest"
```

## Authorization

[rbac.yaml](./testdata/rbac.yaml)

#### Roles
```yaml
kind: Role
metadata:
  name: apiserver-reporter
rules:
  - resources:
      - users 
      - status 
    verbs: ["get", "list", "watch"]
---
kind: Role
metadata:
  name: apiserver-admin
rules:
  - resources:
      - users
      - status 
    verbs: ["*"]
---
kind: Role
metadata:
  name: apiserver-guest
rules:
  - resources:
      - users
      - status 
    nonResourceURLs:
      - "/unauthenticated"
    verbs: ["get", "list", "watch"]
```

#### RoleBinding
```yaml
kind: RoleBinding
metadata:
  name: apiserver-admin
roleRef:
  kind: Role
  name: apiserver-admin
subjects:
  - kind: Group
    name: apiserver:admin
---
kind: RoleBinding
metadata:
  name: apiserver-reporter
roleRef:
  kind: Role
  name: apiserver-reporter
subjects:
  - kind: Group
    name: apiserver:reporter
  - kind: Group
    name: apiserver:admin
---
kind: RoleBinding
metadata:
  name: apiserver-guest
roleRef:
  kind: Role
  name: apiserver-guest
subjects:
  - kind: Group
    name: "*"
```

## Test

```shell
// run server
go run ./apiserver-authorization.go --token-auth-file=./tokens.cvs --authorization-mode=RBAC --rbac-provider=file --rbac-config-path=./testdata

// succeed
curl -X POST http://localhost:8080/api/v1/namespaces/test/users -H "Authorization: Bearer token-admin"
curl -X GET http://localhost:8080/api/v1/namespaces/test/users -H "Authorization: Bearer token-reporter"
curl -X GET http://localhost:8080/api/v1/namespaces/test/status

// failed
curl -X GET http://localhost:8080/api/v1/namespaces/test/users -H "Authorization: Bearer token-guest"
{
  "kind": "Status",
  "apiVersion": "v1",
  "metadata": {},
  "status": "Failure",
  "message": "forbidden: User \"guest\" cannot list resource \"users\" in the namespace \"test\"",
  "reason": "Forbidden",
  "details": {},
  "code": 403
}

curl -X POST http://localhost:8080/api/v1/namespaces/test/users -H "Authorization: Bearer token-reporter"
{
  "kind": "Status",
  "apiVersion": "v1",
  "metadata": {},
  "status": "Failure",
  "message": "forbidden: User \"reporter\" cannot create resource \"users\" in the namespace \"test\"",
  "reason": "Forbidden",
  "details": {},
  "code": 403
}
```

## See Also
- https://kubernetes.io/docs/reference/access-authn-authz/rbac/
