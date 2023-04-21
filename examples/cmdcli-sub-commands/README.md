## Process example

#### Service Mode
```
$go run main.go
I0421 20:18:17.146665   36310 main.go:40] Press ctrl-c to leave the daemon process
^CI0421 20:18:26.683605   36310 proc.go:380] recv shutdown signal, exiting
I0421 20:18:26.683798   36310 proc.go:440] See ya!
```

#### Command Mode
```
$go run main.go m1
I0421 20:18:56.524688   36452 main.go:59] "m1" name="xxx"
```
