## audit

server

```sh
# master
go run ./apiserver-audit-webhook.go

# slave
go run ./apiserver-audit-webhook.go --bind-port 8081 --audit-policy-file ./audit-policy.yaml --audit-webhook-config-file ./audit-webhook.yaml --audit-webhook-batch-max-wait 2s
```



[audit-policy.yaml](./audit-policy.yaml)
```yaml
kind: Policy
rules:
  - level: RequestResponse
    nonResourceURLs:
      - /hello
```

[audit-webhook.yaml](./audit-webhook.yaml)
```
apiVersion: v1
kind: Config
preferences: {}
clusters:
- name: example-cluster
  cluster:
    server: http://127.0.0.1:8080/webhook/audit
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
curl -X POST http://localhost:8081/hello
```

master server stdout

```json
{"metadata":{},"items":[{"level":"RequestResponse","auditID":"a145e3d8-9821-4b48-bdc7-eb9c2a51b7ce","stage":"RequestReceived","requestURI":"/hello","verb":"post","user":{},"sourceIPs":["::1"],"userAgent":"curl/7.64.1","requestReceivedTimestamp":"2021-10-21T05:15:33.930197Z","stageTimestamp":"2021-10-21T05:15:33.930197Z"},{"level":"RequestResponse","auditID":"a145e3d8-9821-4b48-bdc7-eb9c2a51b7ce","stage":"ResponseComplete","requestURI":"/hello","verb":"post","user":{},"sourceIPs":["::1"],"userAgent":"curl/7.64.1","responseStatus":{"metadata":{},"code":200},"requestReceivedTimestamp":"2021-10-21T05:15:33.930197Z","stageTimestamp":"2021-10-21T05:15:33.930354Z","annotations":{"authorization.k8s.io/decision":"allow","authorization.k8s.io/reason":""}}]}
```

