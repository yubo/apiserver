package db

import (
	"context"

	"github.com/yubo/apiserver/pkg/proc"
	v1 "github.com/yubo/apiserver/pkg/proc/api/v1"
	"github.com/yubo/apiserver/pkg/proc/options"
	"github.com/yubo/apiserver/pkg/s3"
	"github.com/yubo/golib/util/validation/field"
)

const (
	moduleName = "s3"
)

type module struct {
	name   string
	client s3.S3Client
}

var (
	_module = &module{name: moduleName}
	hookOps = []v1.HookOps{{
		Hook:        _module.init,
		Owner:       moduleName,
		HookNum:     v1.ACTION_START,
		Priority:    v1.PRI_SYS_INIT,
		SubPriority: v1.PRI_M_DB,
	}}
)

// Because some configuration may be stored in the database,
// set the db.connect into sys.db.prestart
func (p *module) init(ctx context.Context) error {
	cf := s3.NewConfig()
	if err := proc.ReadConfig(p.name, cf); err != nil {
		return field.Invalid(field.NewPath(p.name), cf, err.Error())
	}

	var err error
	if p.client, err = s3.New(cf); err != nil {
		return err
	}

	// set s3 to ctx
	options.WithS3Client(ctx, p.client)

	return nil
}

func Register() {
	proc.RegisterHooks(hookOps)

	proc.AddConfig(moduleName, s3.NewConfig(), proc.WithConfigGroup(moduleName))
}
