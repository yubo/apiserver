package register

import (
	"context"
	"fmt"

	"github.com/yubo/apiserver/pkg/authorization"
	"github.com/yubo/apiserver/pkg/authorization/authorizer"
	"github.com/yubo/apiserver/pkg/options"
	"github.com/yubo/apiserver/plugin/authorizer/rbac/db"
	"github.com/yubo/apiserver/plugin/authorizer/rbac/file"
	"github.com/yubo/golib/configer"
	"github.com/yubo/golib/proc"
	"github.com/yubo/golib/util/errors"
)

const (
	modeName   = "RBAC"
	configPath = "authorization.rbac"
)

type config struct {
	file.Config
	Provider string `json:"provider" flag:"rbac-provider" description:"rbac provider(file,db), used with --authorization-mode=RBAC"`
}

func (o *config) Validate() error {
	allErrors := []error{}

	if o.Provider != "file" && o.Provider != "db" {
		allErrors = append(allErrors, fmt.Errorf("authorization-mode RBAC's authorization --rbac-provider must be set with 'file' or 'db'"))
	}

	if !authorization.IsValidAuthorizationMode(modeName) {
		allErrors = append(allErrors, fmt.Errorf("cannot specify --rbac-provider without mode RBAC"))
	}

	return errors.NewAggregate(allErrors)
}

func newConfig() *config {
	return &config{}
}

func factory(ctx context.Context) (authorizer.Authorizer, error) {
	cf := newConfig()

	if err := configer.ConfigerMustFrom(ctx).Read(configPath, cf); err != nil {
		return nil, err
	}

	switch cf.Provider {
	case "file":
		return file.NewRBAC(&cf.Config)
	case "db":
		return db.NewRBAC(options.DBMustFrom(ctx, ""))
	default:
		return nil, fmt.Errorf("unsupported rbac provider %s", cf.Provider)
	}
}

func init() {
	proc.RegisterFlags(configPath, "authorization", newConfig())
	authorization.RegisterAuthz(modeName, factory)
}
