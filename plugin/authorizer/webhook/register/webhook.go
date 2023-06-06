package register

import (
	"context"
	"errors"
	"fmt"

	"github.com/yubo/apiserver/pkg/authorization"
	"github.com/yubo/apiserver/pkg/authorization/authorizer"
	"github.com/yubo/apiserver/pkg/proc"
	"github.com/yubo/apiserver/plugin/authorizer/webhook"
	"github.com/yubo/golib/api"
	utilerrors "github.com/yubo/golib/util/errors"
	"github.com/yubo/golib/util/wait"
)

const (
	modeName   = "Webhook"
	configPath = "authorization.webhook"
)

type config struct {
	// Kubeconfig file for Webhook authorization plugin.
	WebhookConfigFile string `json:"webhookConfigFile" flag:"authorization-webhook-config-file" description:"File with webhook configuration in kubeconfig format, used with --authorization-mode=Webhook. "`

	// API version of subject access reviews to send to the webhook (e.g. "v1", "v1beta1")
	// WebhookVersion string `json:"webhookVersion" default:"v1beta1" flag:"authorization-webhook-version" description:"The API version of the authorization.k8s.io SubjectAccessReview to send to and expect from the webhook."`

	// TTL for caching of authorized responses from the webhook server.
	WebhookCacheAuthorizedTTL api.Duration `json:"webhookCacheAuthorizedTTL" default:"5m" flag:"authorization-webhook-cache-authorized-ttl" description:"The duration to cache 'authorized' responses from the webhook authorizer."`

	// TTL for caching of unauthorized responses from the webhook server.
	WebhookCacheUnauthorizedTTL api.Duration `json:"webhookCacheUnauthorizedTTL" default:"30s" flag:"authorization-webhook-cache-unauthorized-ttl" description:"The duration to cache 'unauthorized' responses from the webhook authorizer."`

	// WebhookRetryBackoff specifies the backoff parameters for the authorization webhook retry logic.
	// This allows us to configure the sleep time at each iteration and the maximum number of retries allowed
	// before we fail the webhook call in order to limit the fan out that ensues when the system is degraded.
	WebhookRetryBackoff *wait.BackoffConfig `json:"webhookRetryBackoff"`
}

func (o *config) Validate() error {
	allErrors := []error{}

	if o.WebhookConfigFile == "" {
		allErrors = append(allErrors, fmt.Errorf("authorization-mode Webhook's authorization config file not passed"))
	}

	if !authorization.IsValidAuthorizationMode(modeName) {
		allErrors = append(allErrors, fmt.Errorf("cannot specify --authorization-webhook-config-file without mode Webhook"))
	}

	if o.WebhookRetryBackoff != nil && o.WebhookRetryBackoff.Steps <= 0 {
		allErrors = append(allErrors, fmt.Errorf("number of webhook retry attempts must be greater than 1, but is: %d", o.WebhookRetryBackoff.Steps))
	}

	return utilerrors.NewAggregate(allErrors)
}

func newConfig() *config {
	return &config{WebhookRetryBackoff: DefaultAuthWebhookRetryBackoff()}
}

// DefaultAuthWebhookRetryBackoff is the default backoff parameters for
// both authentication and authorization webhook used by the apiserver.
func DefaultAuthWebhookRetryBackoff() *wait.BackoffConfig {
	return &wait.BackoffConfig{
		Duration: api.NewDuration("500ms"),
		Factor:   1.5,
		Jitter:   0.2,
		Steps:    5,
	}
}

func factory(ctx context.Context) (authorizer.Authorizer, error) {
	cf := newConfig()
	if err := proc.ReadConfig(configPath, cf); err != nil {
		return nil, err
	}

	if cf.WebhookRetryBackoff == nil {
		return nil, errors.New("retry backoff parameters for authorization webhook has not been specified")
	}

	return webhook.New(cf.WebhookConfigFile,
		cf.WebhookCacheAuthorizedTTL.Duration,
		cf.WebhookCacheUnauthorizedTTL.Duration,
		cf.WebhookRetryBackoff.Backoff(),
		nil)
}

func init() {
	authorization.RegisterAuthz(modeName, factory)
	proc.AddConfig(configPath, newConfig(), proc.WithConfigGroup("authorization"))
}
