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
  example [flags]


Golib examples flags:

      --city string
                city (env USER_CITY) (default "beijing")
      --license string
                license
      --user-age int
                user age (env USER_AGE)
      --user-name string
                user name (env USER_NAME)

Global flags:
...
```

default setting
```
$ go run ./main.go
=====================================
city: beijing
license: Apache-2.0 license
userAge: 0
userName: Anonymous

=====================================
```

- use --city
```
$ go run ./main.go --city wuhan --user-name steve
=====================================
city: wuhan
license: Apache-2.0 license
userAge: 0
userName: steve

=====================================
```

- use --set-string as yaml.path
```
$ go run ./main.go  --set-string=example.city=wuhan --set-string=example.userName=steve
=====================================
city: wuhan
license: Apache-2.0 license
userAge: 0
userName: steve

=====================================
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
license: Apache-2.0 license
userAge: 16
userName: steve

=====================================
```

#### Environment Variables
```
$ USER_NAME=steve USER_CITY="wuhan" go run ./main.go
=====================================
city: wuhan
license: Apache-2.0 license
userAge: 0
userName: steve

=====================================
```


#### Debug

print as yaml config
```
$ go run ./main.go -f ./config.yaml --debug-config
example:
  city: wuhan
  license: Apache-2.0 license
  userAge: 16
  userName: steve
=====================================
city: wuhan
license: Apache-2.0 license
userAge: 16
userName: steve

=====================================
```

print as flags
```
$ go run ./main.go -f ./config.yaml --debug-flags
Flags:
  --add-dir-header="false"
  --alsologtostderr="false"
  --city="beijing"
  --debug-config="false"
  --debug-flags="true"
  --dry-run="false"
  --help="false"
  --license=""
  --log-backtrace-at=":0"
  --log-dir=""
  --log-file=""
  --log-file-max-size="1800"
  --logtostderr="true"
  --one-output="false"
  --set="[]"
  --set-file="[]"
  --set-string="[]"
  --skip-headers="false"
  --skip-log-headers="false"
  --stderrthreshold="2"
  --user-age="0"
  --user-name="Anonymous"
  --v="0"
  --values="[./config.yaml]"
  --vmodule=""
=====================================
city: wuhan
license: Apache-2.0 license
userAge: 16
userName: steve

=====================================
```
