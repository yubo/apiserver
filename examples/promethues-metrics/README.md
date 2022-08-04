# promethues metrics

https://prometheus.io/docs/guides/go-application/

## server
- config .yaml
```
apiserver:
  enableMetrics: true
```

```
$ go run ./main.go  -f config.yaml
```

## client

```
$ curl -i http://localhost:8080/metrics
```
