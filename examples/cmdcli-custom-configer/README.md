## Process example

#### daemon mode

```sh
$ go run main.go m1
I0421 19:55:51.285985   29931 main.go:68] "m1" name="xxx"
I0421 19:55:51.286218   29931 main.go:69] Press ctrl-c to leave the daemon process
^CI0421 19:55:53.526912   29931 proc.go:380] recv shutdown signal, exiting
I0421 19:55:53.527046   29931 proc.go:439] See ya!
```

#### command line mode

```sh
$ go run main.go m2
I0421 19:56:10.525100   29993 main.go:91] "m2" name="yyy"
I0421 19:56:10.525378   29993 proc.go:439] See ya!
```

#### without process module

```sh
$ go run main.go m3
I0421 19:56:28.986978   30054 main.go:105] "m3" name="zzz"
```

