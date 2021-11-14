## audit

server

```sh
go run ./apiserver-audit.go  --audit-policy-file ./audit-policy.yaml  --audit-log-path -
```

policy
```yaml
kind: Policy
rules:
  - level: None
    nonResourceURLs:
      - /static/*
  - level: RequestResponse
    verbs: ["patch", "delete", "create"]
    nonResourceURLs:
      - /api/user
  - level: Metadata
      - /api/*
```


test

```shell
# None
$ curl -X POST http://localhost:8080/hello

# RequestResponse
$ curl -X POST http://localhost:8080/api/users -d '{"name":"tom", "age": 16}'
{"level":"Metadata","auditID":"09fb2bd8-6281-4a22-a88d-7fa6017f6e1c","stage":"RequestReceived","requestURI":"/api/users","verb":"post","user":{},"sourceIPs":["::1"],"userAgent":"curl/7.64.1","requestReceivedTimestamp":"2021-10-21T05:42:11.210783Z","stageTimestamp":"2021-10-21T05:42:11.210783Z"}
{"level":"Metadata","auditID":"09fb2bd8-6281-4a22-a88d-7fa6017f6e1c","stage":"ResponseComplete","requestURI":"/api/users","verb":"post","user":{},"sourceIPs":["::1"],"userAgent":"curl/7.64.1","responseStatus":{"metadata":{},"code":200},"requestReceivedTimestamp":"2021-10-21T05:42:11.210783Z","stageTimestamp":"2021-10-21T05:42:11.211230Z","annotations":{"authorization.k8s.io/decision":"allow","authorization.k8s.io/reason":""}}


# Metadata
$ curl http://localhost:8080/api/tokens
{"level":"Metadata","auditID":"13af54a7-7ae3-49e8-8dee-a4b7867a9122","stage":"RequestReceived","requestURI":"/api/tokens","verb":"get","user":{},"sourceIPs":["::1"],"userAgent":"curl/7.64.1","requestReceivedTimestamp":"2021-10-21T05:42:40.964340Z","stageTimestamp":"2021-10-21T05:42:40.964340Z"}
{"level":"Metadata","auditID":"13af54a7-7ae3-49e8-8dee-a4b7867a9122","stage":"ResponseComplete","requestURI":"/api/tokens","verb":"get","user":{},"sourceIPs":["::1"],"userAgent":"curl/7.64.1","responseStatus":{"metadata":{},"code":200},"requestReceivedTimestamp":"2021-10-21T05:42:40.964340Z","stageTimestamp":"2021-10-21T05:42:40.964561Z","annotations":{"authorization.k8s.io/decision":"allow","authorization.k8s.io/reason":""}}
```

