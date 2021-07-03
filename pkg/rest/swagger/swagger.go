package swagger

import (
	"context"

	"github.com/yubo/apiserver/pkg/options"
	"github.com/yubo/golib/proc"
	"github.com/yubo/goswagger"
)

const (
	moduleName = "swagger"
)

type Module struct {
	config *goswagger.Config
	name   string
}

var (
	_module = &Module{name: moduleName}
	hookOps = []proc.HookOps{{
		Hook:     _module.init,
		Owner:    moduleName,
		HookNum:  proc.ACTION_START,
		Priority: proc.PRI_MODULE,
	}}
)

func (p *Module) init(ctx context.Context) (err error) {
	c := proc.ConfigerFrom(ctx)

	cf := &goswagger.Config{}
	if err := c.Read(p.name, cf); err != nil {
		return err
	}
	p.config = cf
	// klog.Infof("config %s", c)

	goswagger.New(cf).Install(options.ApiServerMustFrom(ctx))

	return
}

func Register() {
	proc.RegisterHooks(hookOps)
}
