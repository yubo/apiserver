package swagger

import (
	"context"

	"github.com/yubo/apiserver/pkg/options"
	"github.com/yubo/apiserver/pkg/rest"
	"github.com/yubo/golib/configer"
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
	c := configer.ConfigerMustFrom(ctx)

	cf := newConfig()
	if err := c.Read(moduleName, cf); err != nil {
		return err
	}
	p.config = cf

	return goswagger.New(cf).Install(options.APIServerMustFrom(ctx).
		Config().Handler.NonGoRestfulMux.UnlistedHandlePrefix,
		rest.SecuritySchemeRegister)
}

func newConfig() *goswagger.Config {
	return &goswagger.Config{
		Enabled: true,
		Name:    "go-swagger",
		Url:     "/apidocs.json",
	}
}

func Register() {
	proc.RegisterHooks(hookOps)

	proc.RegisterFlags(moduleName, "swagger", newConfig())
}
