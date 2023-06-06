package proc

import (
	"context"
	"os"
	"path/filepath"
	"sync"

	"github.com/go-openapi/spec"
	v1 "github.com/yubo/apiserver/pkg/proc/api/v1"
	"github.com/yubo/golib/configer"
	"github.com/yubo/golib/version"
)

type ProcessOptions struct {
	name        string
	description string
	license     *spec.License
	contact     *spec.ContactInfo
	version     *version.Info

	ctx              context.Context
	cancel           context.CancelFunc
	hooks            []v1.HookOps //  WithHooks
	noloop           bool
	group            bool
	report           bool
	skipLoggingFlags bool
	wg               *sync.WaitGroup
	configerOptions  []configer.ConfigerOption
}

func newProcessOptions() *ProcessOptions {
	ctx, cancel := context.WithCancel(context.Background())

	return &ProcessOptions{
		name:   filepath.Base(os.Args[0]),
		ctx:    ctx,
		cancel: cancel,
		group:  true,
		wg:     &sync.WaitGroup{},
	}
}

type ProcessOption func(*ProcessOptions)

func WithContext(ctx context.Context) ProcessOption {
	return func(p *ProcessOptions) {
		p.ctx, p.cancel = context.WithCancel(ctx)
	}
}

func WithHooks(hooks ...v1.HookOps) ProcessOption {
	return func(p *ProcessOptions) {
		p.hooks = append(p.hooks, hooks...)
	}
}

func WithRegisterAuth(fn ...func(ctx context.Context) error) ProcessOption {
	hooks := make([]v1.HookOps, len(fn))
	for i := range fn {
		hooks[i] = v1.HookOps{
			Hook:        fn[i],
			Owner:       "__proc_with_register_auth__",
			HookNum:     v1.ACTION_START,
			Priority:    v1.PRI_SYS_INIT,
			SubPriority: v1.PRI_M_AUTHN_MODULE,
		}
	}

	return WithHooks(hooks...)
}

func WithRun(fn ...func(ctx context.Context) error) ProcessOption {
	hooks := make([]v1.HookOps, len(fn))
	for i := range fn {
		hooks[i] = v1.HookOps{
			Hook:     fn[i],
			Owner:    "__proc_with_run__",
			HookNum:  v1.ACTION_START,
			Priority: v1.PRI_MODULE,
		}
	}

	return WithHooks(hooks...)
}

func WithName(name string) ProcessOption {
	return func(p *ProcessOptions) {
		p.name = name
	}
}

func WithDescription(description string) ProcessOption {
	return func(p *ProcessOptions) {
		p.description = description
	}
}

func WithLicense(license *spec.License) ProcessOption {
	return func(p *ProcessOptions) {
		p.license = license
	}
}

func WithContact(contact *spec.ContactInfo) ProcessOption {
	return func(p *ProcessOptions) {
		p.contact = contact
	}
}

func WithVersion(version version.Info) ProcessOption {
	return func(p *ProcessOptions) {
		p.version = &version
	}
}

func WithoutLoop() ProcessOption {
	return func(p *ProcessOptions) {
		p.noloop = true
	}
}

func WithoutGroup() ProcessOption {
	return func(p *ProcessOptions) {
		p.group = false
	}
}

func WithWaitGroup(wg *sync.WaitGroup) ProcessOption {
	return func(p *ProcessOptions) {
		p.wg = wg
	}
}

func WithConfigOptions(options ...configer.ConfigerOption) ProcessOption {
	return func(p *ProcessOptions) {
		p.configerOptions = append(p.configerOptions, options...)
	}
}

func WithReport() ProcessOption {
	return func(p *ProcessOptions) {
		p.report = true
	}
}

func WithoutLoggingFlags() ProcessOption {
	return func(p *ProcessOptions) {
		p.skipLoggingFlags = true
	}
}
