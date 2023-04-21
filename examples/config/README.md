## Config example

you can set config by (descending order by priority)

- [Command Line Arguments](#command-line-arguments)
- [YAML file](#yaml-file)
- [Environment Variables](#environment-variables)
- [Debug info](#debug)

#### Command Line Arguments

```
$ go run ./main.go -h


Usage:
  main [flags]

...

Example flags:

      --city string
                city (env USER_CITY) (default "beijing")
      --user-age int
                user age (env USER_AGE)
      --user-name string
                user name (env USER_NAME) (default "Anonymous")
```

default setting
```
$ go run ./main.go
=====================================
city: beijing
userAge: 0
userName: Anonymous

=====================================
I0421 11:19:06.148596   80623 proc.go:403] See ya!
```

- use args
```
$ go run ./main.go --city wuhan --user-name steve
=====================================
city: wuhan
userAge: 0
userName: steve

=====================================
I0421 11:20:17.290832   80791 proc.go:403] See ya!
```

- use --set-string
```
$ go run ./main.go  --set-string=example.city=wuhan --set-string=example.userName=steve
=====================================
city: wuhan
userAge: 0
userName: steve

=====================================
I0421 11:21:12.114467   80891 proc.go:403] See ya!
```

#### YAML file

- [config.yaml](./config.yaml)

```yaml
example:
  userName: steve
  userAge: 16
  city: wuhan
```

```shell
$ go run ./main.go -f ./config.yaml
=====================================
city: wuhan
userAge: 16
userName: steve

=====================================
I0421 11:24:02.570123   81428 proc.go:403] See ya!
```

#### Environment Variables
```
$ USER_NAME=steve USER_CITY="wuhan" go run ./main.go
=====================================
city: wuhan
userAge: 0
userName: steve

=====================================
I0421 11:24:21.990534   81510 proc.go:403] See ya!
```


#### Debug

print as yaml config
```
$ go run ./main.go -f ./config.yaml --debug-config
example:
  city: wuhan
  userAge: 16
  userName: steve
logging:
  flushFrequency: 5s
  format: text
  options:
    json:
      infoBufferSize: "0"
  verbosity: 0
```

print flags
```
$ go run ./main.go -f ./config.yaml -v 1
=====================================
city: wuhan
userAge: 16
userName: steve

=====================================
I0421 13:21:25.314030   89134 flags.go:64] FLAG: --city="beijing"
I0421 13:21:25.314211   89134 flags.go:64] FLAG: --debug-config="false"
I0421 13:21:25.314218   89134 flags.go:64] FLAG: --help="false"
I0421 13:21:25.314221   89134 flags.go:64] FLAG: --log-flush-frequency="5s"
I0421 13:21:25.314226   89134 flags.go:64] FLAG: --log-json-info-buffer-size="0"
I0421 13:21:25.314231   89134 flags.go:64] FLAG: --log-json-split-stream="false"
I0421 13:21:25.314234   89134 flags.go:64] FLAG: --logging-format="text"
I0421 13:21:25.314238   89134 flags.go:64] FLAG: --set="[]"
I0421 13:21:25.314246   89134 flags.go:64] FLAG: --set-file="[]"
I0421 13:21:25.314252   89134 flags.go:64] FLAG: --set-string="[]"
I0421 13:21:25.314259   89134 flags.go:64] FLAG: --user-age="0"
I0421 13:21:25.314263   89134 flags.go:64] FLAG: --user-name="Anonymous"
I0421 13:21:25.314266   89134 flags.go:64] FLAG: --v="1"
I0421 13:21:25.314270   89134 flags.go:64] FLAG: --values="[./config.yaml]"
I0421 13:21:25.314277   89134 flags.go:64] FLAG: --version="false"
I0421 13:21:25.314283   89134 flags.go:64] FLAG: --vmodule=""
I0421 13:21:25.314308   89134 proc.go:403] See ya!
```
