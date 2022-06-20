## Remotecommand - container

#### server

```sh
$ go run ./server/main.go
```


#### client
```sh
$ docker ps
CONTAINER ID   IMAGE                       COMMAND                  CREATED       STATUS      PORTS                    NAMES
e7a59b77821b   prom/node-exporter:latest   "/bin/node_exporter …"   11 days ago   Up 4 days   0.0.0.0:9100->9100/tcp   node-exporter


$ go run ./client/main.go e7a59b77821b uname -a
Linux e7a59b77821b 5.10.76-linuxkit #1 SMP Mon Nov 8 10:21:19 UTC 2021 x86_64 GNU/Linux

```

