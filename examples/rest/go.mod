module examples/rest

go 1.16

replace github.com/yubo/apiserver => ../..

replace github.com/yubo/golib => ../../../golib

require (
	github.com/yubo/apiserver v0.0.0-00010101000000-000000000000
	github.com/yubo/golib v0.0.3-0.20220824085241-a09a630bf3bf
)
