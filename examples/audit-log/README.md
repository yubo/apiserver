## Audit Log


### Server

```sh
$ go run ./main.go -f ./config.yaml
```

### Client

- None

  ```sh
  $ curl -X GET http://localhost:8080/static/hw
  ```


- RequestResponse

  ```sh
  $ curl -X POST http://localhost:8080/api/users -d '{"name":"tom", "age": 16}'
  ```

  server stdout
  ```
  {"level":"RequestResponse","auditID":"90bb94de-c48a-4652-abbd-b069eb25bfb8","stage":"RequestReceived","requestURI":"/api/users","verb":"post","user":{},"sourceIPs":["127.0.0.1"],"userAgent":"curl/7.79.1","requestReceivedTimestamp":"2022-08-10T15:02:55.127020Z","stageTimestamp":"2022-08-10T15:02:55.127020Z"}
  {"level":"RequestResponse","auditID":"90bb94de-c48a-4652-abbd-b069eb25bfb8","stage":"ResponseComplete","requestURI":"/api/users","verb":"post","user":{},"sourceIPs":["127.0.0.1"],"userAgent":"curl/7.79.1","responseStatus":{"metadata":{},"code":200},"requestReceivedTimestamp":"2022-08-10T15:02:55.127020Z","stageTimestamp":"2022-08-10T15:02:55.127330Z","annotations":{"authorization.k8s.io/decision":"allow","authorization.k8s.io/reason":""}}
  ```

- Metadata

  ```sh
  $ curl http://localhost:8080/api/tokens
  ```
  
  server stdout
  ```
  {"level":"Metadata","auditID":"57e62617-fac4-40d6-a12e-c31810b4767b","stage":"RequestReceived","requestURI":"/api/tokens","verb":"get","user":{},"sourceIPs":["127.0.0.1"],"userAgent":"curl/7.79.1","requestReceivedTimestamp":"2022-08-10T15:02:59.070304Z","stageTimestamp":"2022-08-10T15:02:59.070304Z"}
  {"level":"Metadata","auditID":"57e62617-fac4-40d6-a12e-c31810b4767b","stage":"ResponseComplete","requestURI":"/api/tokens","verb":"get","user":{},"sourceIPs":["127.0.0.1"],"userAgent":"curl/7.79.1","responseStatus":{"metadata":{},"code":200},"requestReceivedTimestamp":"2022-08-10T15:02:59.070304Z","stageTimestamp":"2022-08-10T15:02:59.070460Z","annotations":{"authorization.k8s.io/decision":"allow","authorization.k8s.io/reason":""}}
  ```

#### Log level
  - None: don't log events that match this rule.
  - Metadata - log request metadata (requesting user, timestamp, resource, verb, etc.) but not request or response body.
  - Request - log event metadata and request body but not response body. This does not apply for non-resource requests.
  - RequestResponse - log event metadata, request and response bodies. This does not apply for non-resource requests.

## References
  - https://kubernetes.io/docs/tasks/debug-application-cluster/audit/
