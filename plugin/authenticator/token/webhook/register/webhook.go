package webhook

import (
	"context"
	"fmt"
	"time"

	"github.com/yubo/apiserver/pkg/authentication"
	"github.com/yubo/apiserver/pkg/authentication/authenticator"
	"github.com/yubo/golib/proc"
	"github.com/yubo/golib/util/wait"
)

const (
	configPath = "authentication.webhook"
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

func newConfig() *config {
	return &config{
		RetryBackoff: DefaultAuthWebhookRetryBackoff(),
	}
}

func factory(ctx context.Context) (authenticator.Token, error) {
	cf := newConfig()
	if err := proc.ReadConfig(configPath, cf); err != nil {
		return nil, err
	}

	// TODO
	return nil, nil
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
	authentication.RegisterTokenAuthn(factory)
	proc.AddConfig(configPath, newConfig(), proc.WithConfigGroup("authentication"))
}
