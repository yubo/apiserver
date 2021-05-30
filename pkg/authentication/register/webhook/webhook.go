package oidc

import (
	"fmt"
	"time"

	"github.com/spf13/pflag"
	"github.com/yubo/apiserver/pkg/options"
	"github.com/yubo/golib/proc"
	pconfig "github.com/yubo/golib/proc/config"
	"github.com/yubo/golib/staging/util/wait"
	"github.com/yubo/golib/util"
)

const (
	moduleName       = "authentication.webhook"
	noUsernamePrefix = "-"
)

var (
	_auth   = &authModule{name: moduleName}
	hookOps = []proc.HookOps{{
		Hook:        _auth.init,
		Owner:       moduleName,
		HookNum:     proc.ACTION_START,
		Priority:    proc.PRI_SYS_INIT - 1,
		SubPriority: options.PRI_M_AUTHN,
	}}
	_config *config
)

type config struct {
	ConfigFile string
	Version    string
	CacheTTL   time.Duration

	// RetryBackoff specifies the backoff parameters for the authentication webhook retry logic.
	// This allows us to configure the sleep time at each iteration and the maximum number of retries allowed
	// before we fail the webhook call in order to limit the fan out that ensues when the system is degraded.
	RetryBackoff *wait.Backoff
}

func (o *config) addFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.ConfigFile, "authentication-token-webhook-config-file", o.ConfigFile, ""+
		"File with webhook configuration for token authentication in kubeconfig format. "+
		"The API server will query the remote service to determine authentication for bearer tokens.")

	fs.StringVar(&o.Version, "authentication-token-webhook-version", o.Version, ""+
		"The API version of the authentication.k8s.io TokenReview to send to and expect from the webhook.")

	fs.DurationVar(&o.CacheTTL, "authentication-token-webhook-cache-ttl", o.CacheTTL,
		"The duration to cache responses from the webhook token authenticator.")

}

func (o *config) changed() interface{} {
	if o == nil {
		return nil
	}
	return util.Diff2Map(defaultConfig(), o)
}

func (o *config) Validate() error {
	retryBackoff := o.RetryBackoff
	if retryBackoff != nil && retryBackoff.Steps <= 0 {
		return fmt.Errorf("number of webhook retry attempts must be greater than 1, but is: %d", retryBackoff.Steps)
	}
	return nil
}

type authModule struct {
	name   string
	config *config
}

func defaultConfig() *config {
	return &config{
		Version:      "v1beta1",
		CacheTTL:     2 * time.Minute,
		RetryBackoff: DefaultAuthWebhookRetryBackoff(),
	}
}

func (p *authModule) init(ops *proc.HookOps) error {
	configer := ops.Configer()

	cf := defaultConfig()
	if err := configer.ReadYaml(p.name, cf,
		pconfig.WithOverride(_config.changed())); err != nil {
		return err
	}
	p.config = cf

	// TODO
	return nil
}

// DefaultAuthWebhookRetryBackoff is the default backoff parameters for
// both authentication and authorization webhook used by the apiserver.
func DefaultAuthWebhookRetryBackoff() *wait.Backoff {
	return &wait.Backoff{
		Duration: 500 * time.Millisecond,
		Factor:   1.5,
		Jitter:   0.2,
		Steps:    5,
	}
}

func init() {
	proc.RegisterHooks(hookOps)
	_config = defaultConfig()
	_config.addFlags(proc.NamedFlagSets().FlagSet("authentication"))
}
