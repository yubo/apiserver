module github.com/yubo/apiserver

go 1.13

require (
	github.com/coreos/go-oidc v2.1.0+incompatible
	github.com/creack/pty v1.1.11
	github.com/emicklei/go-restful v2.15.0+incompatible
	github.com/emicklei/go-restful-openapi v1.4.1
	github.com/fortytw2/leaktest v1.3.0
	github.com/go-openapi/spec v0.20.3
	github.com/golang/protobuf v1.5.2
	github.com/google/go-cmp v0.5.5
	github.com/google/uuid v1.1.2
	github.com/gorilla/websocket v1.4.2
	github.com/json-iterator/go v1.1.11
	github.com/modern-go/reflect2 v1.0.1
	github.com/opentracing-contrib/go-stdlib v1.0.0
	github.com/opentracing/opentracing-go v1.2.0
	github.com/prometheus/client_golang v1.9.0
	github.com/spf13/cobra v1.1.1
	github.com/stretchr/testify v1.6.1
	github.com/uber-go/tally v3.3.17+incompatible
	github.com/uber/jaeger-client-go v2.25.0+incompatible
	github.com/uber/jaeger-lib v2.2.0+incompatible
	github.com/yubo/golib v0.0.0-20210530151551-3c17bab5028d
	github.com/yubo/goswagger v0.0.0-20210530151208-4cd1816f5cf6
	go.uber.org/zap v1.13.0
	golang.org/x/net v0.0.0-20210525063256-abc453219eb5
	golang.org/x/sync v0.0.0-20201020160332-67f06af15bc9
	google.golang.org/grpc v1.27.1
	google.golang.org/protobuf v1.26.0
	gopkg.in/square/go-jose.v2 v2.2.2
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/apimachinery v0.21.2 // indirect
	k8s.io/klog v1.0.0
	k8s.io/klog/v2 v2.9.0
	sigs.k8s.io/yaml v1.2.0
)

replace (
	github.com/yubo/golib => ../golib
	github.com/yubo/goswagger => ../goswagger
)
