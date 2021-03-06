/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package authenticatorfactory

import (
	"errors"
	"time"

	"github.com/yubo/apiserver/pkg/authentication/authenticator"
	"github.com/yubo/apiserver/pkg/authentication/group"
	"github.com/yubo/apiserver/pkg/authentication/request/anonymous"
	unionauth "github.com/yubo/apiserver/pkg/authentication/request/union"
	// "github.com/yubo/apiserver/pkg/authentication/request/x509"
)

// DelegatingAuthenticatorConfig is the minimal configuration needed to create an authenticator
// built to delegate authentication to a kube API server
type DelegatingAuthenticatorConfig struct {
	Anonymous bool

	// TokenAccessReviewClient is a client to do token review. It can be nil. Then every token is ignored.
	//TokenAccessReviewClient authenticationclient.TokenReviewInterface

	// WebhookRetryBackoff specifies the backoff parameters for the authentication webhook retry logic.
	// This allows us to configure the sleep time at each iteration and the maximum number of retries allowed
	// before we fail the webhook call in order to limit the fan out that ensues when the system is degraded.
	//WebhookRetryBackoff *wait.Backoff

	// CacheTTL is the length of time that a token authentication answer will be cached.
	CacheTTL time.Duration

	// CAContentProvider are the options for verifying incoming connections using mTLS and directly assigning to users.
	// Generally this is the CA bundle file used to authenticate client certificates
	// If this is nil, then mTLS will not be used.
	//ClientCertificateCAContentProvider CAContentProvider

	APIAudiences authenticator.Audiences
}

func (c DelegatingAuthenticatorConfig) New() (authenticator.Request /*, *spec.SecurityDefinitions*/, error) {
	authenticators := []authenticator.Request{}
	//securityDefinitions := spec.SecurityDefinitions{}

	// front-proxy first, then remote
	// Add the front proxy authenticator if requested

	// x509 client cert auth
	//if c.ClientCertificateCAContentProvider != nil {
	//	authenticators = append(authenticators, x509.NewDynamic(c.ClientCertificateCAContentProvider.VerifyOptions, x509.CommonNameUserConversion))
	//}

	//if c.TokenAccessReviewClient != nil {
	//	if c.WebhookRetryBackoff == nil {
	//		return nil, errors.New("retry backoff parameters for delegating authentication webhook has not been specified")
	//	}
	//	tokenAuth, err := webhooktoken.NewFromInterface(c.TokenAccessReviewClient, c.APIAudiences, *c.WebhookRetryBackoff)
	//	if err != nil {
	//		return nil, err
	//	}
	//	cachingTokenAuth := cache.New(tokenAuth, false, c.CacheTTL, c.CacheTTL)
	//	authenticators = append(authenticators, bearertoken.New(cachingTokenAuth), websocket.NewProtocolAuthenticator(cachingTokenAuth))

	//	//securityDefinitions["BearerToken"] = &spec.SecurityScheme{
	//	//	SecuritySchemeProps: spec.SecuritySchemeProps{
	//	//		Type:        "apiKey",
	//	//		Name:        "authorization",
	//	//		In:          "header",
	//	//		Description: "Bearer Token authentication",
	//	//	},
	//	//}
	//}

	if len(authenticators) == 0 {
		if c.Anonymous {
			return anonymous.NewAuthenticator(), nil
		}
		return nil, errors.New("No authentication method configured")
	}

	authenticator := group.NewAuthenticatedGroupAdder(unionauth.New(authenticators...))
	if c.Anonymous {
		authenticator = unionauth.NewFailOnError(authenticator, anonymous.NewAuthenticator())
	}
	return authenticator, nil
}
