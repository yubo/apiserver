package logging

import (
	"context"
	"fmt"

	"github.com/yubo/apiserver/components/logs"
	logsapi "github.com/yubo/apiserver/components/logs/api/v1"
	procapi "github.com/yubo/apiserver/pkg/proc/api/v1"
	v1 "github.com/yubo/apiserver/pkg/proc/api/v1"
	utilfeature "github.com/yubo/apiserver/pkg/util/feature"
	"github.com/yubo/golib/configer"
	"github.com/yubo/golib/util"
	"github.com/yubo/golib/util/runtime"
	"github.com/yubo/golib/util/validation/field"
)

//func Register() {
//	RegisterHooks(hookOps)
//	AddConfig(moduleName, newConfig(), WithConfigGroup("logging"))
//}

func init() {
	runtime.Must(logsapi.AddFeatureGates(utilfeature.DefaultMutableFeatureGate))
}

const (
	ModuleName = "logging"
)

var (
	_module = &logging{name: ModuleName}
	HookOps = []v1.HookOps{{
		Hook:        _module.start,
		Owner:       "logging",
		HookNum:     procapi.ACTION_START,
		Priority:    procapi.PRI_SYS_INIT,
		SubPriority: procapi.PRI_M_LOGGING,
	}}
)

type LoggingConfig struct {
	*logsapi.LoggingConfiguration
}

func NewConfig() *LoggingConfig {
	return &LoggingConfig{
		LoggingConfiguration: logsapi.NewLoggingConfiguration(),
	}
}

func (p LoggingConfig) String() string {
	return util.Prettify(p)
}

func (p *LoggingConfig) Validate() error {
	return nil
}

type logging struct {
	name   string
	config *logsapi.LoggingConfiguration
}

func (p *logging) start(ctx context.Context) error {
	if err := logsapi.AddFeatureGates(utilfeature.DefaultMutableFeatureGate); err != nil {
		return err
	}

	// Config and flags parsed, now we can initialize logging.
	logs.InitLogs()

	config := NewConfig()
	if err := configer.ConfigerMustFrom(ctx).Read(p.name, config); err != nil {
		return err
	}

	if err := logsapi.ValidateAndApplyAsField(config.LoggingConfiguration, utilfeature.DefaultFeatureGate, field.NewPath("logging")); err != nil {
		return fmt.Errorf("initialize logging: %v", err)
	}

	return nil
}
