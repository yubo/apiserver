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
      - /hello/*
  - level: RequestResponse
    verbs: ["patch", "delete", "create"]
    resources:
  - level: Metadata
```


test

```
curl -X POST http://localhost:8080/hello
```

stdout

```json
{"level":"Metadata","auditID":"09ba5922-0e9c-44cb-a600-139df7437ce3","stage":"RequestReceived","requestURI":"/hello","verb":"post","user":{},"sourceIPs":["::1"],"userAgent":"curl/7.29.0","requestReceivedTimestamp":"2021-10-19T18:43:06.711056Z","stageTimestamp":"2021-10-19T18:43:06.711056Z"}
{"level":"Metadata","auditID":"09ba5922-0e9c-44cb-a600-139df7437ce3","stage":"ResponseComplete","requestURI":"/hello","verb":"post","user":{},"sourceIPs":["::1"],"userAgent":"curl/7.29.0","responseStatus":{"metadata":{},"code":200},"requestReceivedTimestamp":"2021-10-19T18:43:06.711056Z","stageTimestamp":"2021-10-19T18:43:06.711258Z","annotations":{"authorization.k8s.io/decision":"allow","authorization.k8s.io/reason":""}}

```

