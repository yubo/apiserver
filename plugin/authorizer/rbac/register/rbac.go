package register

import (
	"context"
	"fmt"

	"github.com/yubo/apiserver/pkg/authorization"
	"github.com/yubo/apiserver/pkg/authorization/authorizer"
	"github.com/yubo/apiserver/plugin/authorizer/rbac/db"
	"github.com/yubo/apiserver/plugin/authorizer/rbac/file"
	"github.com/yubo/apiserver/pkg/proc"
	"github.com/yubo/golib/util/errors"
	"k8s.io/klog/v2"
)

const (
	modeName   = "RBAC"
	configPath = "authorization.rbac"
)

type config struct {
	file.Config
	//Provider string `json:"provider" flag:"rbac-provider" description:"rbac provider(file,db), used with --authorization-mode=RBAC"`
}

func (o *config) Validate() error {
	allErrors := []error{}

	//if o.Provider != "file" && o.Provider != "db" {
	//	allErrors = append(allErrors, fmt.Errorf("authorization-mode RBAC's authorization --rbac-provider must be set with 'file' or 'db'"))
	//}

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

	if err := proc.ReadConfig(configPath, cf); err != nil {
		return nil, err
	}

	if cf.Config.ConfigPath != "" {
		return file.NewRBAC(&cf.Config)
	}

	klog.Info("use ")

	// if not set file, try find rbac provider from storage
	return db.NewRBAC()
}

func init() {
	authorization.RegisterAuthz(modeName, factory)
	proc.AddConfig(configPath, newConfig(), proc.WithConfigGroup("authorization"))
}
