package webhook

import (
	"context"
	"fmt"
	"time"

	"github.com/yubo/apiserver/pkg/options"
	"github.com/yubo/golib/proc"
	"github.com/yubo/golib/staging/util/wait"
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
		Priority:    proc.PRI_SYS_INIT,
		SubPriority: options.PRI_M_AUTHN - 1,
	}}
	_config *config
)

type config struct {
	ConfigFile string `json:"configFile" flag:"authentication-token-webhook-config-file" description:"File with webhook configuration for token authentication in kubeconfig format. The API server will query the remote service to determine authentication for bearer tokens."`
	Version    string `json:"version" default:"v1beta1" flag:"authentication-token-webhook-version" description:"The API version of the authentication.k8s.io TokenReview to send to and expect from the webhook."`
	CacheTTL   int    `json:"cacheTTL" default:"120s" flag:"authentication-token-webhook-cache-ttl" description:"The duration to cache responses from the webhook token authenticator."`
	cacheTTL   time.Duration

	// RetryBackoff specifies the backoff parameters for the authentication webhook retry logic.
	// This allows us to configure the sleep time at each iteration and the maximum number of retries allowed
	// before we fail the webhook call in order to limit the fan out that ensues when the system is degraded.
	RetryBackoff *wait.Backoff `json:"-"`
}

func (o *config) Validate() error {
	retryBackoff := o.RetryBackoff
	if retryBackoff != nil && retryBackoff.Steps <= 0 {
		return fmt.Errorf("number of webhook retry attempts must be greater than 1, but is: %d", retryBackoff.Steps)
	}

	o.cacheTTL = time.Duration(o.CacheTTL) * time.Second

	return nil
}

type authModule struct {
	name   string
	config *config
}

func newConfig() *config {
	return &config{
		RetryBackoff: DefaultAuthWebhookRetryBackoff(),
	}
}

func (p *authModule) init(ctx context.Context) error {
	c := proc.ConfigerFrom(ctx)

	cf := newConfig()
	if err := c.Read(p.name, cf); err != nil {
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
	proc.RegisterFlags(moduleName, "authentication", newConfig())
}
