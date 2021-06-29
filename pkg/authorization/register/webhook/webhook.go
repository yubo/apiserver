package webhook

import (
	"fmt"
	"time"

	"github.com/yubo/apiserver/pkg/authorization"
	"github.com/yubo/apiserver/pkg/authorization/authorizer"
	"github.com/yubo/apiserver/pkg/options"
	"github.com/yubo/golib/proc"
	utilerrors "github.com/yubo/golib/staging/util/errors"
	"github.com/yubo/golib/staging/util/wait"
)

const (
	moduleName       = "authorization"
	submoduleName    = "Webhook"
	noUsernamePrefix = "-"
)

var (
	_auth   = &authModule{name: moduleName + "." + submoduleName}
	hookOps = []proc.HookOps{{
		Hook:        _auth.init,
		Owner:       moduleName,
		HookNum:     proc.ACTION_START,
		Priority:    proc.PRI_SYS_INIT,
		SubPriority: options.PRI_M_AUTHZ - 1,
	}}
	_config *config
)

type config struct {

	// Kubeconfig file for Webhook authorization plugin.
	WebhookConfigFile string `json:"webhookConfigFile" flag:"authorization-webhook-config-file" description:"File with webhook configuration in kubeconfig format, used with --authorization-mode=Webhook. "`

	// API version of subject access reviews to send to the webhook (e.g. "v1", "v1beta1")
	WebhookVersion string `json:"webhookVersion" default:"v1beta1" flag:"authorization-webhook-version" description:"The API version of the authorization.k8s.io SubjectAccessReview to send to and expect from the webhook."`

	// TTL for caching of authorized responses from the webhook server.
	WebhookCacheAuthorizedTTL time.Duration `json:"webhookCacheAuthorizedTTL" default:"5m" flag:"authorization-webhook-cache-authorized-ttl" description:"The duration to cache 'authorized' responses from the webhook authorizer."`

	// TTL for caching of unauthorized responses from the webhook server.
	WebhookCacheUnauthorizedTTL time.Duration `json:"webhookCacheUnauthorizedTTL" default:"30s" flag:"authorization-webhook-cache-unauthorized-ttl" description:"The duration to cache 'unauthorized' responses from the webhook authorizer."`

	// WebhookRetryBackoff specifies the backoff parameters for the authorization webhook retry logic.
	// This allows us to configure the sleep time at each iteration and the maximum number of retries allowed
	// before we fail the webhook call in order to limit the fan out that ensues when the system is degraded.
	WebhookRetryBackoff *wait.Backoff `json:"webhookRetryBackoff"`
}

func (o *config) Validate() error {
	allErrors := []error{}

	if o.WebhookConfigFile == "" {
		return nil
	}

	if o.WebhookConfigFile == "" {
		allErrors = append(allErrors, fmt.Errorf("authorization-mode Webhook's authorization config file not passed"))
	}

	if o.WebhookRetryBackoff != nil && o.WebhookRetryBackoff.Steps <= 0 {
		allErrors = append(allErrors, fmt.Errorf("number of webhook retry attempts must be greater than 1, but is: %d", o.WebhookRetryBackoff.Steps))
	}

	return utilerrors.NewAggregate(allErrors)
}

type authModule struct {
	name   string
	config *config
}

func newConfig() *config {
	return &config{WebhookRetryBackoff: DefaultAuthWebhookRetryBackoff()}
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

func (p *authModule) init(ops *proc.HookOps) error {
	c := ops.Configer()

	cf := newConfig()
	if err := c.ReadYaml(moduleName, cf); err != nil {
		return err
	}
	p.config = cf

	return nil
}

func init() {
	proc.RegisterHooks(hookOps)
	proc.RegisterFlags(moduleName, "authorization", newConfig())

	factory := func() (authorizer.Authorizer, error) {
		//cf := _auth.config
		return nil, nil

	}

	authorization.RegisterAuthz(submoduleName, factory)

}
