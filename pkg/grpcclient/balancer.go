package grpcclient

import (
	"fmt"
	"math/rand"
	"strings"
	"sync"

	"github.com/yubo/apiserver/pkg/config/configgrpc"
	"google.golang.org/grpc/balancer/roundrobin"
	"google.golang.org/grpc/resolver"
)

var (
	grpcScheme      = "grpc_balancer_scheme"
	resolverBuilder = lbResolverBuilder{
		scheme:     grpcScheme,
		addrsStore: make(map[string][]string),
	}
)

func init() {
	resolver.Register(&resolverBuilder)
}

func prepareConfigWithBalancer(in *configgrpc.GRPCClientSettings) (*configgrpc.GRPCClientSettings, error) {
	cf := in.Copy()

	addrs := strings.Split(cf.Endpoint, ",")
	if len(addrs) < 2 {
		return cf, nil
	}

	if !configgrpc.ValidateBalancerName(cf.BalancerName) {
		return nil, fmt.Errorf("invalid balancer_name: %s with Endpoint: %s", cf.BalancerName, cf.Endpoint)
	}

	if cf.BalancerName == roundrobin.Name {
		randSlice(addrs)
	}

	serviceName := cf.Endpoint
	resolverBuilder.addService(serviceName, addrs)
	cf.Endpoint = fmt.Sprintf("%s:///%s", grpcScheme, serviceName)

	return cf, nil
}

func randSlice(in []string) {
	size := len(in)
	if size < 1 {
		return
	}

	for i := 0; i < size-1; i++ {
		//addr[size-i] <-> [0, size-i)
		src := size - i - 1
		dst := rand.Intn(src + 1)

		t := in[src]
		in[src] = in[dst]
		in[dst] = t
	}
}

type lbResolverBuilder struct {
	sync.RWMutex
	scheme     string
	addrsStore map[string][]string
}

func (p *lbResolverBuilder) addService(serviceName string, addrs []string) {
	p.Lock()
	defer p.Unlock()

	p.addrsStore[serviceName] = addrs
}

func (p *lbResolverBuilder) getService(serviceName string) []string {
	p.RLock()
	defer p.RUnlock()

	return p.addrsStore[serviceName]

}

func (p *lbResolverBuilder) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	r := &lbResolver{
		lbResolverBuilder: p,
		target:            target,
		cc:                cc,
	}
	r.start()
	return r, nil
}

func (p *lbResolverBuilder) Scheme() string { return p.scheme }

type lbResolver struct {
	*lbResolverBuilder
	target resolver.Target
	cc     resolver.ClientConn
}

func (r *lbResolver) start() {
	addrStrs := r.getService(r.target.Endpoint)
	addrs := make([]resolver.Address, len(addrStrs))
	for i, s := range addrStrs {
		addrs[i] = resolver.Address{Addr: s}
	}
	r.cc.UpdateState(resolver.State{Addresses: addrs})
}
func (*lbResolver) ResolveNow(o resolver.ResolveNowOptions) {}
func (*lbResolver) Close()                                  {}
