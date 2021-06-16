/*
Copyright 2017 The Kubernetes Authors.

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

/*
Package bootstrap provides a token authenticator for TLS bootstrap secrets.
*/
package bootstrap

import (
	"context"
	"crypto/subtle"
	"fmt"
	"time"

	"k8s.io/klog/v2"

	"github.com/yubo/apiserver/pkg/authentication/authenticator"
	"github.com/yubo/apiserver/pkg/authentication/user"
	corev1 "github.com/yubo/apiserver/pkg/api/core/v1"
	"github.com/yubo/golib/staging/api/errors"
	//corev1listers "k8s.io/client-go/listers/core/v1"
	corev1listers "github.com/yubo/apiserver/pkg/listers"
	bootstrapapi "github.com/yubo/apiserver/staging/cluster-bootstrap/token/api"
	bootstrapsecretutil "github.com/yubo/apiserver/staging/cluster-bootstrap/util/secrets"
	bootstraptokenutil "github.com/yubo/apiserver/staging/cluster-bootstrap/util/tokens"
)

// TODO: A few methods in this package is copied from other sources. Either
// because the existing functionality isn't exported or because it is in a
// package that shouldn't be directly imported by this packages.

// NewTokenAuthenticator initializes a bootstrap token authenticator.
//
// Lister is expected to be for the "kube-system" namespace.
func NewTokenAuthenticator(lister corev1listers.SecretLister) *TokenAuthenticator {
	return &TokenAuthenticator{lister}
}

// TokenAuthenticator authenticates bootstrap tokens from secrets in the API server.
type TokenAuthenticator struct {
	lister corev1listers.SecretLister
}

// tokenErrorf prints a error message for a secret that has matched a bearer
// token but fails to meet some other criteria.
//
//    tokenErrorf(secret, "has invalid value for key %s", key)
//
func tokenErrorf(s *corev1.Secret, format string, i ...interface{}) {
	format = fmt.Sprintf("Bootstrap secret %s/%s matching bearer token ", s.Namespace, s.Name) + format
	klog.V(3).Infof(format, i...)
}

// AuthenticateToken tries to match the provided token to a bootstrap token secret
// in a given namespace. If found, it authenticates the token in the
// "system:bootstrappers" group and with the "system:bootstrap:(token-id)" username.
//
// All secrets must be of type "bootstrap.kubernetes.io/token". An example secret:
//
//     apiVersion: v1
//     kind: Secret
//     metadata:
//       # Name MUST be of form "bootstrap-token-( token id )".
//       name: bootstrap-token-( token id )
//       namespace: kube-system
//     # Only secrets of this type will be evaluated.
//     type: bootstrap.kubernetes.io/token
//     data:
//       token-secret: ( private part of token )
//       token-id: ( token id )
//       # Required key usage.
//       usage-bootstrap-authentication: true
//       auth-extra-groups: "system:bootstrappers:custom-group1,system:bootstrappers:custom-group2"
//       # May also contain an expiry.
//
// Tokens are expected to be of the form:
//
//     ( token-id ).( token-secret )
//
func (t *TokenAuthenticator) AuthenticateToken(ctx context.Context, token string) (*authenticator.Response, bool, error) {
	tokenID, tokenSecret, err := bootstraptokenutil.ParseToken(token)
	if err != nil {
		// Token isn't of the correct form, ignore it.
		return nil, false, nil
	}

	secretName := bootstrapapi.BootstrapTokenSecretPrefix + tokenID
	secret, err := t.lister.Get(secretName)
	if err != nil {
		if errors.IsNotFound(err) {
			klog.V(3).Infof("No secret of name %s to match bootstrap bearer token", secretName)
			return nil, false, nil
		}
		return nil, false, err
	}

	if secret.DeletionTimestamp != nil {
		tokenErrorf(secret, "is deleted and awaiting removal")
		return nil, false, nil
	}

	if string(secret.Type) != string(bootstrapapi.SecretTypeBootstrapToken) || secret.Data == nil {
		tokenErrorf(secret, "has invalid type, expected %s.", bootstrapapi.SecretTypeBootstrapToken)
		return nil, false, nil
	}

	ts := bootstrapsecretutil.GetData(secret, bootstrapapi.BootstrapTokenSecretKey)
	if subtle.ConstantTimeCompare([]byte(ts), []byte(tokenSecret)) != 1 {
		tokenErrorf(secret, "has invalid value for key %s, expected %s.", bootstrapapi.BootstrapTokenSecretKey, tokenSecret)
		return nil, false, nil
	}

	id := bootstrapsecretutil.GetData(secret, bootstrapapi.BootstrapTokenIDKey)
	if id != tokenID {
		tokenErrorf(secret, "has invalid value for key %s, expected %s.", bootstrapapi.BootstrapTokenIDKey, tokenID)
		return nil, false, nil
	}

	if bootstrapsecretutil.HasExpired(secret, time.Now()) {
		// logging done in isSecretExpired method.
		return nil, false, nil
	}

	if bootstrapsecretutil.GetData(secret, bootstrapapi.BootstrapTokenUsageAuthentication) != "true" {
		tokenErrorf(secret, "not marked %s=true.", bootstrapapi.BootstrapTokenUsageAuthentication)
		return nil, false, nil
	}

	groups, err := bootstrapsecretutil.GetGroups(secret)
	if err != nil {
		tokenErrorf(secret, "has invalid value for key %s: %v.", bootstrapapi.BootstrapTokenExtraGroupsKey, err)
		return nil, false, nil
	}

	return &authenticator.Response{
		User: &user.DefaultInfo{
			Name:   bootstrapapi.BootstrapUserPrefix + string(id),
			Groups: groups,
		},
	}, true, nil
}

func (a *TokenAuthenticator) Name() string {
	return "bearer OIDC authenticator"
}

func (a *TokenAuthenticator) Priority() int {
	return authenticator.PRI_TOKEN_BOOTSTRAP
}

func (a *TokenAuthenticator) Available() bool {
	return true
}
