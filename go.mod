module github.com/yubo/apiserver

go 1.15

require (
	github.com/Azure/go-autorest/autorest v0.11.19
	github.com/Azure/go-autorest/autorest/adal v0.9.14
	github.com/buger/goterm v1.0.1
	github.com/containerd/containerd v1.5.5 // indirect
	github.com/coreos/go-oidc v2.1.0+incompatible
	github.com/creack/pty v1.1.11
	github.com/docker/distribution v2.7.1+incompatible
	github.com/docker/docker v17.12.0-ce-rc1.0.20200916142827-bd33bbf0497b+incompatible
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/elazarl/goproxy/ext v0.0.0-20210801061803-8e322dfb79c4 // indirect
	github.com/emicklei/go-restful v2.15.0+incompatible
	github.com/emicklei/go-restful-openapi v1.4.1
	github.com/fortytw2/leaktest v1.3.0
	github.com/go-openapi/spec v0.20.3
	github.com/golang/mock v1.4.1
	github.com/golang/protobuf v1.5.2
	github.com/google/go-cmp v0.5.5
	github.com/google/gofuzz v1.2.0
	github.com/google/uuid v1.2.0
	github.com/gorilla/websocket v1.4.2
	github.com/moby/term v0.0.0-20201216013528-df9cb8a40635
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822
	github.com/opencontainers/go-digest v1.0.0
	github.com/opentracing-contrib/go-stdlib v1.0.0
	github.com/opentracing/opentracing-go v1.2.0
	github.com/pquerna/cachecontrol v0.1.0 // indirect
	github.com/prometheus/client_golang v1.9.0
	github.com/prometheus/procfs v0.7.1 // indirect
	github.com/spf13/cobra v1.1.1
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.7.0
	github.com/uber-go/tally v3.3.17+incompatible
	github.com/uber/jaeger-client-go v2.25.0+incompatible
	github.com/uber/jaeger-lib v2.2.0+incompatible
	github.com/yubo/golib v0.0.0-20210729083123-4040286093c6
	github.com/yubo/goswagger v0.0.0-20210729084640-1f6e710ffaaf
	go.uber.org/zap v1.13.0
	golang.org/x/net v0.0.0-20210525063256-abc453219eb5
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	golang.org/x/sync v0.0.0-20201207232520-09787c993a3a
	google.golang.org/grpc v1.33.2
	google.golang.org/protobuf v1.26.0
	gopkg.in/square/go-jose.v2 v2.5.1
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.22.0
	k8s.io/apimachinery v0.22.0
	k8s.io/client-go v0.22.0
	k8s.io/klog/v2 v2.9.0
)

replace (
	github.com/docker/distribution => github.com/docker/distribution v2.7.1+incompatible
	github.com/docker/docker => github.com/docker/docker v17.12.0-ce-rc1.0.20200916142827-bd33bbf0497b+incompatible
	github.com/docker/go-connections => github.com/docker/go-connections v0.4.0
	github.com/docker/go-units => github.com/docker/go-units v0.4.0
	github.com/docker/spdystream => github.com/docker/spdystream v0.0.0-20160310174837-449fdfce4d96
	github.com/yubo/golib => ../golib
	google.golang.org/grpc => google.golang.org/grpc v1.27.1
)
