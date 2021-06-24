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

package anonymous

import (
	"net/http"

	"github.com/yubo/apiserver/pkg/authentication/authenticator"
	"github.com/yubo/apiserver/pkg/authentication/user"
)

const (
	anonymousUser = user.Anonymous

	unauthenticatedGroup = user.AllUnauthenticated
)

type Authenticator struct{}

func NewAuthenticator() authenticator.Request {
	return &Authenticator{}
}

func (a *Authenticator) AuthenticateRequest(req *http.Request) (*authenticator.Response, bool, error) {
	auds, _ := authenticator.AudiencesFrom(req.Context())
	return &authenticator.Response{
		User: &user.DefaultInfo{
			Name:   anonymousUser,
			Groups: []string{unauthenticatedGroup},
		},
		Audiences: auds,
	}, true, nil
}

func (a *Authenticator) Name() string {
	return "anonymous authenticator"
}

func (a *Authenticator) Priority() int {
	return authenticator.PRI_TOKEN_OIDC
}

func (a *Authenticator) Available() bool {
	return true
}
