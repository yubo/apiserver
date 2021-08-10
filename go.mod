module github.com/yubo/apiserver

go 1.13

replace github.com/yubo/golib => ../golib

require (
	github.com/buger/goterm v1.0.1
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
	github.com/matttproud/golang_protobuf_extensions v1.0.2-0.20181231171920-c182affec369 // indirect
	github.com/modern-go/reflect2 v1.0.1
	github.com/opentracing-contrib/go-stdlib v1.0.0
	github.com/opentracing/opentracing-go v1.2.0
	github.com/pquerna/cachecontrol v0.1.0 // indirect
	github.com/prometheus/client_golang v1.9.0
	github.com/prometheus/procfs v0.7.1 // indirect
	github.com/spf13/cobra v1.1.1
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.6.1
	github.com/uber-go/tally v3.3.17+incompatible
	github.com/uber/jaeger-client-go v2.25.0+incompatible
	github.com/uber/jaeger-lib v2.2.0+incompatible
	github.com/yubo/golib v0.0.0-20210729083123-4040286093c6
	github.com/yubo/goswagger v0.0.0-20210729084640-1f6e710ffaaf
	go.uber.org/zap v1.13.0
	golang.org/x/crypto v0.0.0-20210220033148-5ea612d1eb83 // indirect
	golang.org/x/net v0.0.0-20210525063256-abc453219eb5
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d // indirect
	golang.org/x/sync v0.0.0-20201207232520-09787c993a3a
	google.golang.org/appengine v1.6.5 // indirect
	google.golang.org/grpc v1.27.1
	google.golang.org/protobuf v1.26.0
	gopkg.in/square/go-jose.v2 v2.2.2
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/klog/v2 v2.9.0
	sigs.k8s.io/yaml v1.2.0
)
