package proc

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/go-openapi/spec"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	cliflag "github.com/yubo/apiserver/components/cli/flag"

	"github.com/yubo/apiserver/components/cli/globalflag"
	"github.com/yubo/apiserver/components/featuregate"
	"github.com/yubo/apiserver/components/logs"
	"github.com/yubo/apiserver/components/version/verflag"
	v1 "github.com/yubo/apiserver/pkg/proc/api/v1"
	"github.com/yubo/apiserver/pkg/proc/logging"
	"github.com/yubo/apiserver/pkg/proc/reporter"
	"github.com/yubo/golib/api/errors"
	"github.com/yubo/golib/configer"
	"github.com/yubo/golib/util"
	"github.com/yubo/golib/version"
	"k8s.io/klog/v2"
)

const (
	serverGracefulCloseTimeout = 12 * time.Second
	moduleName                 = "proc"
)

var (
	DefaultProcess = NewProcess()
)

type Process struct {
	*ProcessOptions
	featureGate featuregate.MutableFeatureGate

	configer       configer.Configer
	parsedConfiger configer.ParsedConfiger
	//fs             *pflag.FlagSet

	// config
	configs       []*configOptions // catalog of RegisterConfig
	namedFlagSets cliflag.NamedFlagSets

	debugConfig bool // print config after proc.init()

	hookOps [v1.ACTION_SIZE]v1.Hooks // catalog of RegisterHooks
	status  v1.ProcessStatus
	err     error

	addFlagsOnce sync.Once
}

func newProcess() *Process {
	hookOps := [v1.ACTION_SIZE]v1.Hooks{}
	for i := v1.ACTION_START; i < v1.ACTION_SIZE; i++ {
		hookOps[i] = v1.Hooks{}
	}

	return &Process{
		hookOps:        hookOps,
		ProcessOptions: newProcessOptions(),
		configer:       configer.New(),
		featureGate:    featuregate.NewFeatureGate(),
	}

}

func NewProcess(opts ...ProcessOption) *Process {
	p := newProcess()

	for _, opt := range opts {
		opt(p.ProcessOptions)
	}

	return p
}

func Context() context.Context {
	return DefaultProcess.Context()
}

func NewRootCmd(opts ...ProcessOption) *cobra.Command {
	return DefaultProcess.NewRootCmd(opts...)
}

func Start(fs *pflag.FlagSet) error {
	return DefaultProcess.Start(fs)
}

func Init(cmd *cobra.Command, opts ...ProcessOption) error {
	DefaultProcess.Init(cmd, opts...)
	return nil
}

func Shutdown() error {
	return DefaultProcess.shutdown()
}

func PrintConfig(w io.Writer) {
	DefaultProcess.PrintConfig(w)
}

func PrintFlags(fs *pflag.FlagSet) {
	DefaultProcess.PrintFlags(fs)
}

func AddFlags(name string) {
	DefaultProcess.AddFlags(name)
}

func Name() string {
	return DefaultProcess.Name()
}

func Description() string {
	return DefaultProcess.Description()
}

func License() *spec.License {
	return DefaultProcess.License()
}
func Contact() *spec.ContactInfo {
	return DefaultProcess.Contact()
}
func Version() *version.Info {
	return DefaultProcess.Version()
}

func NamedFlagSets() *cliflag.NamedFlagSets {
	return &DefaultProcess.namedFlagSets
}

func NewVersionCmd() *cobra.Command {
	return DefaultProcess.NewVersionCmd()
}

func RegisterHooks(in []v1.HookOps) error {
	return DefaultProcess.RegisterHooks(in)
}

func Configer() configer.ParsedConfiger {
	return DefaultProcess.parsedConfiger
}

func ReadConfig(path string, into interface{}) error {
	err := DefaultProcess.parsedConfiger.Read(path, into)
	if err == nil {
		return nil
	}
	if _, ok := err.(configer.ErrNoValue); ok {
		return nil
	}
	return err
}

func MustReadConfig(path string, into interface{}) error {
	err := DefaultProcess.parsedConfiger.Read(path, into)
	if err == nil {
		return nil
	}
	if _, ok := err.(configer.ErrNoValue); ok {
		return errors.NewNotFound(path)
	}
	return err
}

func AddConfig(path string, sample interface{}, opts ...ConfigOption) error {
	return DefaultProcess.AddConfig(path, sample, opts...)
}

func AddGlobalConfig(sample interface{}) error {
	return DefaultProcess.AddConfig("", sample, WithConfigGroup("global"))
}

func Parse(fs *pflag.FlagSet, opts ...configer.ConfigerOption) (configer.ParsedConfiger, error) {
	return DefaultProcess.Parse(fs, opts...)
}

// RegisterHooks register hookOps as a module
func (p *Process) RegisterHooks(in []v1.HookOps) error {
	for i := range in {
		v := &in[i]
		v.SetContext(p.ctx)
		//v.process = p
		//v.priority = v1.ProcessPriority(uint32(v.Priority)<<(16-3) + uint32(v.SubPriority))

		p.hookOps[v.HookNum] = append(p.hookOps[v.HookNum], v)
	}
	return nil
}

// with proc.Start
func (p *Process) NewRootCmd(opts ...ProcessOption) *cobra.Command {
	rand.Seed(time.Now().UnixNano())
	runtime.GOMAXPROCS(runtime.NumCPU())

	for _, opt := range opts {
		opt(p.ProcessOptions)
	}

	cmd := &cobra.Command{
		Use:          p.name,
		Short:        p.description,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			verflag.PrintAndExitIfRequested()
			return p.Start(cmd.Flags())
		},
		Args: func(cmd *cobra.Command, args []string) error {
			for _, arg := range args {
				if len(arg) > 0 {
					return fmt.Errorf("%q does not take any arguments, got %q", cmd.CommandPath(), args)
				}
			}
			return nil
		},
	}

	p.Init(cmd)

	return cmd
}

// NewCmd without set runtime
func (p *Process) NewCmd(opts ...ProcessOption) *cobra.Command {
	for _, opt := range opts {
		opt(p.ProcessOptions)
	}

	cmd := &cobra.Command{
		Use:          p.name,
		Short:        p.description,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			verflag.PrintAndExitIfRequested()
			return p.Start(cmd.Flags())
		},
		Args: func(cmd *cobra.Command, args []string) error {
			for _, arg := range args {
				if len(arg) > 0 {
					return fmt.Errorf("%q does not take any arguments, got %q", cmd.CommandPath(), args)
				}
			}
			return nil
		},
	}

	p.Init(cmd)

	return cmd
}

func (p *Process) Context() context.Context {
	return p.ctx
}

func (p *Process) Start(fs *pflag.FlagSet) error {
	// To help debugging, immediately log version
	klog.Infof("Version: %+v", p.version)

	klog.InfoS("Golang settings", "GOGC", os.Getenv("GOGC"), "GOMAXPROCS", os.Getenv("GOMAXPROCS"), "GOTRACEBACK", os.Getenv("GOTRACEBACK"))

	if _, err := p.Parse(fs); err != nil {
		return err
	}

	if p.debugConfig {
		p.PrintConfig(os.Stdout)
		os.Exit(0)
	}

	if err := p.start(); err != nil {
		return err
	}

	p.PrintFlags(fs)

	if p.noloop {
		p.stop()
		return p.err
	}

	if p.report {
		if err := reporter.NewBuildReporter(p.ctx, p.version).Start(); err != nil {
			return err
		}
	}

	if _, err := SdNotify(true, "READY=1\n"); err != nil {
		klog.Errorf("Unable to send systemd daemon successful start message: %v\n", err)
	}

	<-SetupSignalHandler()

	p.stop()

	if err := p.err; err != nil {
		return err
	}

	klog.V(1).Info("[graceful-termination] apiserver is exiting")
	return nil
}

func (p *Process) Parse(fs *pflag.FlagSet, opts ...configer.ConfigerOption) (configer.ParsedConfiger, error) {
	// parse configpositive
	if p.parsedConfiger == nil {
		opts = append(p.configerOptions, opts...)
		c, err := p.configer.Parse(opts...)
		if err != nil {
			return nil, err
		}
		p.parsedConfiger = c
	}

	return p.parsedConfiger, nil
}

// Init
// set configer options
// alloc p.ctx
// validate config each module
// sort hook options
func (p *Process) Init(cmd *cobra.Command, opts ...ProcessOption) error {
	for _, opt := range opts {
		opt(p.ProcessOptions)
	}

	if err := p.RegisterHooks(p.hooks); err != nil {
		return err
	}

	// check custom configer
	if c, ok := configer.ConfigerFrom(p.ctx); ok {
		p.parsedConfiger = c
	}

	if _, ok := AttrFrom(p.ctx); !ok {
		p.ctx = WithAttr(p.ctx, make(map[interface{}]interface{}))
	}
	withWg(p.ctx, p.wg)

	// globalflags
	p.AddFlags(cmd.Name())

	fs := cmd.PersistentFlags()

	// bind logging flags
	//if err := logsapi.Init(fs); err != nil {
	//	return fmt.Errorf("proc.log.init %s", err)
	//}

	// bind flags
	fs.ParseErrorsWhitelist.UnknownFlags = true
	if err := p.BindRegisteredFlags(fs); err != nil {
		return fmt.Errorf("proc.BindRegisteredFlags %s", err)
	}

	if p.group {
		setGroupCommandFunc(cmd, p.namedFlagSets)
	}

	return nil
}

// only be called once
func (p *Process) start() error {
	p.wg.Add(1)
	defer p.wg.Done()

	for i := v1.ACTION_START; i < v1.ACTION_SIZE; i++ {
		p.hookOps[i].Sort()
	}

	// start
	ctx := configer.WithConfiger(p.ctx, p.parsedConfiger)
	for _, ops := range p.hookOps[v1.ACTION_START] {
		ops.Dlog()
		if err := ops.Hook(WithHookOps(ctx, ops)); err != nil {
			return fmt.Errorf("%s.%s() err: %s", ops.Owner, util.Name(ops.Hook), err)
		}
	}
	p.status.Set(v1.STATUS_RUNNING)
	return nil
}

func (p *Process) shutdown() error {
	proc, err := os.FindProcess(os.Getpid())
	if err != nil {
		return err
	}
	return proc.Signal(shutdownSignal)
}

// reverse order
func (p *Process) stop() error {
	select {
	case <-p.ctx.Done():
		return nil
	default:
	}

	// cancel ctx first
	p.cancel()
	p.status.Set(v1.STATUS_EXIT)

	wg := p.wg
	wgCh := make(chan struct{})

	wg.Add(1)
	go func() {
		defer wg.Done()

		ops := p.hookOps[v1.ACTION_STOP]
		ctx := configer.WithConfiger(p.ctx, p.parsedConfiger)
		for i := len(ops) - 1; i >= 0; i-- {
			op := ops[i]
			op.Dlog()
			if err := op.Hook(WithHookOps(ctx, op)); err != nil {
				p.err = fmt.Errorf("%s.%s() err: %s", op.Owner, util.Name(op.Hook), err)
				return
			}
		}
	}()

	go func() {
		wg.Wait()
		wgCh <- struct{}{}
	}()

	// Wait then close or hard close.
	closeTimeout := serverGracefulCloseTimeout
	select {
	case <-wgCh:
		if !p.noloop {
			klog.Info("See ya!")
		}
	case <-time.After(closeTimeout):
		p.err = fmt.Errorf("%s closed after timeout %s", p.name, closeTimeout.String())
	}

	return p.err
}

func (p *Process) PrintConfig(out io.Writer) {
	out.Write([]byte(p.parsedConfiger.String()))
}

func (p *Process) PrintFlags(fs *pflag.FlagSet) {
	cliflag.PrintFlags(fs)
}

//func (p *Process) AddLoggingFlags() {
//	fs := p.namedFlagSets.FlagSet("logging")
//	o := logsapi.NewLoggingConfiguration()
//	logsapi.AddFlags(o, fs)
//	logs.AddFlags(fs, logs.SkipLoggingConfigurationFlags())
//}

func (p *Process) AddFlags(name string) {
	p.addFlagsOnce.Do(func() {
		gfs := p.namedFlagSets.FlagSet("global")
		opts := []logs.Option{}
		if p.skipLoggingFlags {
			opts = append(opts, logs.SkipLoggingConfigurationFlags())
		}

		// add klog, logs, help flags
		globalflag.AddGlobalFlags(
			gfs,
			name,
			opts...,
		)

		// add version flags
		verflag.AddFlags(gfs)

		// add process flags to gfs
		gfs.BoolVar(&p.debugConfig, "debug-config", p.debugConfig, "print config")
		p.configer.AddFlags(gfs)
	})
}

func (p *Process) Name() string {
	return p.name
}

func (p *Process) Description() string {
	return p.description
}
func (p *Process) License() *spec.License {
	return p.license
}
func (p *Process) Contact() *spec.ContactInfo {
	return p.contact
}
func (p *Process) Version() *version.Info {
	return p.version
}

func (p *Process) NewVersionCmd() *cobra.Command {
	ver := p.version
	if ver == nil {
		panic(v1.InvalidVersionErr)
	}

	return &cobra.Command{
		Use:   "version",
		Short: "show version, git commit",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("Git Version:     %s\n", ver.GitVersion)
			fmt.Printf("Git Commit:      %s\n", ver.GitCommit)
			fmt.Printf("Git Tree State:  %s\n", ver.GitTreeState)
			fmt.Printf("Build Date:      %s\n", ver.BuildDate)
			fmt.Printf("Go Version:      %s\n", ver.GoVersion)
			fmt.Printf("Compiler:        %s\n", ver.Compiler)
			fmt.Printf("Platform:        %s\n", ver.Platform)
			return nil
		},
	}
}

func init() {
	RegisterHooks(logging.HookOps)
	AddConfig(logging.ModuleName, logging.NewConfig(), WithConfigGroup("logging"))
}
