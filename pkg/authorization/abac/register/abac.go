package register

import (
	"context"
	"fmt"

	"github.com/yubo/apiserver/pkg/authorization"
	"github.com/yubo/apiserver/pkg/authorization/abac"
	"github.com/yubo/apiserver/pkg/authorization/abac/api"
	"github.com/yubo/apiserver/pkg/authorization/authorizer"
	"github.com/yubo/golib/proc"
	"github.com/yubo/golib/util/errors"
)

const (
	modeName   = "ABAC"
	configPath = "authorization"
)

var (
	PolicyList []*api.Policy
)

type config struct {
	PolicyFile string `json:"policyFile" flag:"authorization-policy-file" description:"File with authorization policy in json line by line format, used with --authorization-mode=ABAC, on the secure port."`
}

func (o *config) Validate() error {
	allErrors := []error{}

	if o.PolicyFile == "" {
		allErrors = append(allErrors, fmt.Errorf("authorization-mode ABAC's authorization config file not passed"))

	}

	if !authorization.IsValidAuthorizationMode(modeName) {
		allErrors = append(allErrors, fmt.Errorf("cannot specify --authorization-policy-file without mode ABAC"))
	}

	return errors.NewAggregate(allErrors)
}

func newConfig() *config {
	return &config{}
}

func factory(ctx context.Context) (authorizer.Authorizer, error) {
	cf := newConfig()
	if err := proc.ConfigerMustFrom(ctx).Read(configPath, cf); err != nil {
		return nil, err
	}

	p, err := abac.NewFromFile(cf.PolicyFile)
	if err != nil {
		return nil, err
	}
	return abac.PolicyList(append(PolicyList, p...)), nil
}

func init() {
	proc.RegisterFlags(configPath, "authorization", newConfig())
	authorization.RegisterAuthz(modeName, factory)
}
