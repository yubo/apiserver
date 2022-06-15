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
	CAContentProvider dynamiccertificates.CAContentProvider `json:"-"`
}

func (s *config) Validate() error {
	return nil
}

// GetClientVerifyOptionFn provides verify options for your authenticator while respecting the preferred order of verifiers.
func (s *config) GetClientCAContentProvider() (dynamiccertificates.CAContentProvider, error) {
	if s.CAContentProvider != nil {
		return s.CAContentProvider, nil
	}

	if len(s.ClientCA) == 0 {
		return nil, nil
	}

	return dynamiccertificates.NewDynamicCAContentFromFile("client-ca-bundle", s.ClientCA)
}

func newConfig() *config { return &config{} }

func factory(ctx context.Context) (authenticator.Request, error) {
	cf := newConfig()
	if err := proc.ReadConfig(configPath, cf); err != nil {
		return nil, err
	}

	if cf.ClientCA == "" {
		klog.V(5).Infof("authnModule x509 ignore", "reason", "clientCAFile is not set")
		return nil, nil
	}

	servingInfo := options.APIServerMustFrom(ctx).Config().SecureServing
	if servingInfo == nil {
		klog.V(5).InfoS("authnModule x509 ignore", "reason", "servingInfo was not found")
		return nil, nil
	}

	klog.V(5).InfoS("authnModule x509", "ca file", cf.ClientCA)

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
	authentication.RegisterAuthn(factory)
	proc.AddConfig(configPath, newConfig(), proc.WithConfigGroup("authentication"))
}
