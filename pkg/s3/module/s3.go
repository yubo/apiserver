package db

import (
	"context"

	"github.com/yubo/apiserver/pkg/options"
	"github.com/yubo/apiserver/pkg/s3"
	"github.com/yubo/golib/proc"
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
	hookOps = []proc.HookOps{{
		Hook:        _module.init,
		Owner:       moduleName,
		HookNum:     proc.ACTION_START,
		Priority:    proc.PRI_SYS_INIT,
		SubPriority: options.PRI_M_DB,
	}}
)

// Because some configuration may be stored in the database,
// set the db.connect into sys.db.prestart
func (p *module) init(ctx context.Context) (err error) {
	cf := newConfig()
	if err := proc.ReadConfig(p.name, cf); err != nil {
		return err
	}

	if p.client, err = s3.New(cf); err != nil {
		return err
	}

	options.WithS3Client(ctx, p.client)

	return nil
}

func newConfig() *s3.Config {
	return &s3.Config{}
}

func Register() {
	proc.RegisterHooks(hookOps)

	proc.AddConfig(moduleName, newConfig(), proc.WithConfigGroup(moduleName))
}
