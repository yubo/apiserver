# Example

This directory includes example logger setup allowing users to easily check and test impact of logging configuration. 

Below we can see examples of how some features work.

## Default

Run:
```console
go run ./main.go
```

Expected output:
```
I0420 15:21:59.331177   10677 logger.go:39] Log using Infof, key: value
I0420 15:21:59.331432   10677 logger.go:40] "Log using InfoS" key="value"
E0420 15:21:59.331438   10677 logger.go:42] Log using Errorf, err: fail
E0420 15:21:59.331443   10677 logger.go:43] "Log using ErrorS" err="fail"
I0420 15:21:59.331447   10677 logger.go:45] Log with sensitive key, data: {"secret"}
```

## JSON 

Run:
```console
go run ./logger.go --logging-format json
```

Expected output:
```
yubo@yubo-didi-mbp16 ~/src/apiserver/examples/logging [main]$ go run ./main.go --logging-format json
{"ts":1682045883117.218,"caller":"logging/main.go:25","msg":"Log using Infof, key: value\n","v":0}
{"ts":1682045883117.257,"caller":"logging/main.go:26","msg":"Log using InfoS","v":0,"key":"value"}
{"ts":1682045883117.269,"caller":"logging/main.go:28","msg":"Log using Errorf, err: fail\n"}
{"ts":1682045883117.275,"caller":"logging/main.go:29","msg":"Log using ErrorS","err":"fail"}
{"ts":1682045883117.2869,"caller":"logging/main.go:31","msg":"Log with sensitive key, data: {\"secret\"}\n","v":0}
{"ts":1682045883117.334,"caller":"proc/proc.go:403","msg":"See ya!\n","v":0}
```

## Verbosity

```console
go run ./main.go -v1
```

```
I0421 10:56:58.830662   76406 main.go:25] Log using Infof, key: value
I0421 10:56:58.830862   76406 main.go:26] "Log using InfoS" key="value"
E0421 10:56:58.830869   76406 main.go:28] Log using Errorf, err: fail
E0421 10:56:58.830874   76406 main.go:29] "Log using ErrorS" err="fail"
I0421 10:56:58.830877   76406 main.go:31] Log with sensitive key, data: {"secret"}
I0421 10:56:58.830883   76406 main.go:32] Log less important message
I0421 10:56:58.830900   76406 flags.go:65] FLAG: --v="1"
I0421 10:56:58.830920   76406 proc.go:403] See ya!
```

The last line is not printed at the default log level.
