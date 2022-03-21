## audit

server

```sh
# master
go run ./apiserver-audit-webhook.go

# slave
go run ./apiserver-audit-webhook.go --port 8081 --audit-policy-file ./audit-policy.yaml --audit-webhook-config-file ./audit-webhook.yaml --audit-webhook-batch-max-wait 2s
```



[audit-policy.yaml](./audit-policy.yaml)
```yaml
kind: Policy
rules:
  - level: RequestResponse
    nonResourceURLs:
      - /audit/hello
```

[audit-webhook.yaml](./audit-webhook.yaml)
```
apiVersion: v1
kind: Config
preferences: {}
clusters:
- name: example-cluster
  cluster:
    server: http://127.0.0.1:8080/audit/webhook
users:
- name: example-user
  user:
    username: some-user
    password: some-password
contexts:
- name: example-context
  context:
    cluster: example-cluster
    user: example-user
current-context: example-context
```


test

```
curl -X POST http://localhost:8081/audit/hello
```

master server stdout

```json
{"metadata":{},"items":[{"level":"RequestResponse","auditID":"655dbb82-57e0-400d-9980-8cd7f19bf884","stage":"RequestReceived","requestURI":"/audit/hello","verb":"post","user":{},"sourceIPs":["::1"],"userAgent":"curl/7.77.0","requestReceivedTimestamp":"2022-03-21T07:57:53.382469Z","stageTimestamp":"2022-03-21T07:57:53.382469Z"},{"level":"RequestResponse","auditID":"655dbb82-57e0-400d-9980-8cd7f19bf884","stage":"ResponseComplete","requestURI":"/audit/hello","verb":"post","user":{},"sourceIPs":["::1"],"userAgent":"curl/7.77.0","responseStatus":{"metadata":{},"code":200},"requestReceivedTimestamp":"2022-03-21T07:57:53.382469Z","stageTimestamp":"2022-03-21T07:57:53.382641Z","annotations":{"authorization.k8s.io/decision":"allow","authorization.k8s.io/reason":""}}]}
```

