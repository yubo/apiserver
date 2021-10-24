package x509

import (
	"context"

	"github.com/yubo/apiserver/pkg/authentication"
	"github.com/yubo/apiserver/pkg/authentication/authenticator"
	"github.com/yubo/apiserver/pkg/authentication/request/x509"
	"github.com/yubo/apiserver/pkg/dynamiccertificates"
	"github.com/yubo/apiserver/pkg/options"
	"github.com/yubo/golib/proc"
	"k8s.io/klog/v2"
)

const (
	moduleName = "authentication.x509"
	configPath = "authentication"
)

type config struct {
	// ClientCA is the certificate bundle for all the signers that you'll recognize for incoming client certificates
	ClientCA string `json:"clientCAFile" flag:"client-ca-file" description:"If set, any request presenting a client certificate signed by one of the authorities in the client-ca-file is authenticated with an identity corresponding to the CommonName of the client certificate."`

	// CAContentProvider are the options for verifying incoming connections using mTLS and directly assigning to users.
	// Generally this is the CA bundle file used to authenticate client certificates
	// If non-nil, this takes priority over the ClientCA file.
	// CAContentProvider dynamiccertificates.CAContentProvider ``
}

func (s *config) Validate() error {
	return nil
}

func newConfig() *config { return &config{} }

func factory(ctx context.Context) (authenticator.Request, error) {
	c := proc.ConfigerMustFrom(ctx)

	cf := newConfig()
	if err := c.Read(configPath, cf); err != nil {
		return nil, err
	}

	if cf.ClientCA == "" {
		klog.V(5).Infof("authnModule x509 ignore", "reason", "clientCAFile is not set")
		return nil, nil
	}

	servingInfo := options.APIServerMustFrom(ctx).ServingInfo()
	if servingInfo == nil || servingInfo.ClientCA == nil {
		klog.V(5).InfoS("authnModule x509 ignore", "reason", "clientCA was not found")
		return nil, nil
	}

	clientCA, err := dynamiccertificates.NewDynamicCAContentFromFile("client-ca-bundle", cf.ClientCA)
	if err != nil {
		return nil, err
	}

	if err := servingInfo.ApplyClientCert(clientCA); err != nil {
		return nil, err
	}

	return x509.NewDynamic(servingInfo.ClientCA.VerifyOptions, x509.CommonNameUserConversion), nil
}

func init() {
	proc.RegisterFlags(configPath, "authentication", newConfig())
	authentication.RegisterAuthn(factory)
}
