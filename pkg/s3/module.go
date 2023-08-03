package s3

import (
	"context"

	"github.com/yubo/apiserver/components/dbus"
	"github.com/yubo/apiserver/pkg/proc"
	v1 "github.com/yubo/apiserver/pkg/proc/api/v1"
	"github.com/yubo/golib/util/validation/field"
)

const moduleName = "s3"

func Register(opts ...proc.ModuleOption) {
	o := &proc.ModuleOptions{
		Proc: proc.DefaultProcess,
	}
	for _, v := range opts {
		v(o)
	}

	m := &module{name: moduleName}
	hookOps := []v1.HookOps{{
		Hook:        m.init,
		Owner:       moduleName,
		HookNum:     v1.ACTION_START,
		Priority:    v1.PRI_SYS_INIT,
		SubPriority: v1.PRI_M_DB,
	}}

	o.Proc.RegisterHooks(hookOps)
	o.Proc.AddConfig(moduleName, NewConfig(), proc.WithConfigGroup(moduleName))
}

type module struct {
	name   string
	client S3Client
}

// Because some configuration may be stored in the database,
// set the db.connect into sys.db.prestart
func (p *module) init(ctx context.Context) error {
	cf := NewConfig()
	if err := proc.ReadConfig(p.name, cf); err != nil {
		return field.Invalid(field.NewPath(p.name), cf, err.Error())
	}

	var err error
	if p.client, err = New(cf); err != nil {
		return err
	}

	// set s3 to ctx
	dbus.RegisterS3Client(p.client)

	return nil
}
