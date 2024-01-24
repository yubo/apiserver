package proc

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"

	"github.com/go-openapi/spec"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/yubo/apiserver/components/cli/flag"
	cliflag "github.com/yubo/apiserver/components/cli/flag"
	v1 "github.com/yubo/apiserver/pkg/proc/api/v1"
	"github.com/yubo/golib/configer"
	"github.com/yubo/golib/term"
	"github.com/yubo/golib/version"
)

var (
	DefaultProcess = NewProcess()
)

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
	return DefaultProcess.ReadConfig(path, into)
}

func MustReadConfig(path string, into interface{}) error {
	return DefaultProcess.MustReadConfig(path, into)
}

func AddConfig(path string, sample interface{}, opts ...ConfigOption) error {
	return DefaultProcess.AddConfig(path, sample, opts...)
}

func AddGlobalConfig(sample interface{}) error {
	return DefaultProcess.AddGlobalConfig(sample)
}

func Parse(fs *pflag.FlagSet, opts ...configer.ConfigerOption) (configer.ParsedConfiger, error) {
	return DefaultProcess.Parse(fs, opts...)
}
func PrintErrln(err error) int {
	if err == nil {
		return 0
	}

	fmt.Fprintf(os.Stderr, "Error: %s\n", err)
	return 1
}

func envOr(name string, defs ...string) string {
	if v, ok := os.LookupEnv(name); ok {
		return v
	}
	for _, def := range defs {
		if def != "" {
			return def
		}
	}
	return ""
}

func getenvBool(str string) bool {
	b, _ := strconv.ParseBool(os.Getenv(str))
	return b
}

func sigContains(v os.Signal, sigs []os.Signal) bool {
	for _, sig := range sigs {
		if sig == v {
			return true
		}
	}
	return false
}

//func NameOfFunction(f interface{}) string {
//	fun := runtime.FuncForPC(reflect.ValueOf(f).Pointer())
//	tokenized := strings.Split(fun.Name(), ".")
//	last := tokenized[len(tokenized)-1]
//	last = strings.TrimSuffix(last, ")·fm") // < Go 1.5
//	last = strings.TrimSuffix(last, ")-fm") // Go 1.5
//	last = strings.TrimSuffix(last, "·fm")  // < Go 1.5
//	last = strings.TrimSuffix(last, "-fm")  // Go 1.5
//	return last
//}

func setGroupCommandFunc(cmd *cobra.Command, nfs flag.NamedFlagSets) {
	usageFmt := "Usage:\n  %s\n"
	cols, _, _ := term.GetTerminalSize(cmd.OutOrStdout())
	cmd.SetUsageFunc(func(cmd *cobra.Command) error {
		fmt.Fprintf(cmd.OutOrStderr(), usageFmt, cmd.UseLine())
		flag.PrintSections(cmd.OutOrStderr(), nfs, cols)
		return nil
	})
	cmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		fmt.Fprintf(cmd.OutOrStdout(), "%s\n\n"+usageFmt, cmd.Long, cmd.UseLine())
		flag.PrintSections(cmd.OutOrStdout(), nfs, cols)
	})
}

// SdNotify sends a message to the init daemon. It is common to ignore the error.
// If `unsetEnvironment` is true, the environment variable `NOTIFY_SOCKET`
// will be unconditionally unset.
//
// It returns one of the following:
// (false, nil) - notification not supported (i.e. NOTIFY_SOCKET is unset)
// (false, err) - notification supported, but failure happened (e.g. error connecting to NOTIFY_SOCKET or while sending data)
// (true, nil) - notification supported, data has been sent
func SdNotify(unsetEnvironment bool, state string) (bool, error) {
	socketAddr := &net.UnixAddr{
		Name: os.Getenv("NOTIFY_SOCKET"),
		Net:  "unixgram",
	}

	// NOTIFY_SOCKET not set
	if socketAddr.Name == "" {
		return false, nil
	}

	if unsetEnvironment {
		if err := os.Unsetenv("NOTIFY_SOCKET"); err != nil {
			return false, err
		}
	}

	conn, err := net.DialUnix(socketAddr.Net, nil, socketAddr)
	// Error connecting to NOTIFY_SOCKET
	if err != nil {
		return false, err
	}
	defer conn.Close()

	if _, err = conn.Write([]byte(state)); err != nil {
		return false, err
	}
	return true, nil
}
