// pkg/kubeapiserver/authorizer/config.go
package authorization

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/pflag"
	authzmodes "github.com/yubo/apiserver/pkg/authorization/modes"
	utilerrors "github.com/yubo/golib/staging/util/errors"
	"github.com/yubo/golib/staging/util/sets"
	"github.com/yubo/golib/staging/util/wait"
	"github.com/yubo/golib/util"
	// "github.com/yubo/apiserver/pkg/authorization/rbac"
)

// config contains all build-in authorization options for API Server
type config struct {
	Modes                       []string
	PolicyFile                  string
	WebhookConfigFile           string
	WebhookVersion              string
	WebhookCacheAuthorizedTTL   time.Duration
	WebhookCacheUnauthorizedTTL time.Duration
	// WebhookRetryBackoff specifies the backoff parameters for the authorization webhook retry logic.
	// This allows us to configure the sleep time at each iteration and the maximum number of retries allowed
	// before we fail the webhook call in order to limit the fan out that ensues when the system is degraded.
	WebhookRetryBackoff *wait.Backoff
}

// newConfig create a config with default value
func newConfig() *config {
	return &config{
		Modes:                       []string{authzmodes.ModeAlwaysAllow},
		WebhookVersion:              "v1beta1",
		WebhookCacheAuthorizedTTL:   5 * time.Minute,
		WebhookCacheUnauthorizedTTL: 30 * time.Second,
		WebhookRetryBackoff:         DefaultAuthWebhookRetryBackoff(),
	}
}
func (o *config) Changed() interface{} {
	if o == nil {
		return nil
	}
	return util.Diff2Map(newConfig(), o)
}
func (o *config) String() string {
	return util.Prettify(o)
}

// Validate checks invalid config combination
func (o *config) Validate() error {
	if o == nil {
		return nil
	}
	allErrors := []error{}

	if len(o.Modes) == 0 {
		allErrors = append(allErrors, fmt.Errorf("at least one authorization-mode must be passed"))
	}

	modes := sets.NewString(o.Modes...)
	for _, mode := range o.Modes {
		if !authzmodes.IsValidAuthorizationMode(mode) {
			allErrors = append(allErrors, fmt.Errorf("authorization-mode %q is not a valid mode", mode))
		}
		if mode == authzmodes.ModeABAC {
			if o.PolicyFile == "" {
				allErrors = append(allErrors, fmt.Errorf("authorization-mode ABAC's authorization policy file not passed"))
			}
		}
		if mode == authzmodes.ModeWebhook {
			if o.WebhookConfigFile == "" {
				allErrors = append(allErrors, fmt.Errorf("authorization-mode Webhook's authorization config file not passed"))
			}
		}
	}

	if o.PolicyFile != "" && !modes.Has(authzmodes.ModeABAC) {
		allErrors = append(allErrors, fmt.Errorf("cannot specify --authorization-policy-file without mode ABAC"))
	}

	if o.WebhookConfigFile != "" && !modes.Has(authzmodes.ModeWebhook) {
		allErrors = append(allErrors, fmt.Errorf("cannot specify --authorization-webhook-config-file without mode Webhook"))
	}

	if len(o.Modes) != len(modes.List()) {
		allErrors = append(allErrors, fmt.Errorf("authorization-mode %q has mode specified more than once", o.Modes))
	}

	if o.WebhookRetryBackoff != nil && o.WebhookRetryBackoff.Steps <= 0 {
		allErrors = append(allErrors, fmt.Errorf("number of webhook retry attempts must be greater than 1, but is: %d", o.WebhookRetryBackoff.Steps))
	}

	return utilerrors.NewAggregate(allErrors)
}

// addFlags returns flags of authorization for a API Server
func (o *config) addFlags(fs *pflag.FlagSet) {
	fs.StringSliceVar(&o.Modes, "authorization-mode", o.Modes, ""+
		"Ordered list of plug-ins to do authorization on secure port. Comma-delimited list of: "+
		strings.Join(authzmodes.AuthorizationModeChoices, ",")+".")

	fs.StringVar(&o.PolicyFile, "authorization-policy-file", o.PolicyFile, ""+
		"File with authorization policy in json line by line format, used with --authorization-mode=ABAC, on the secure port.")

	fs.StringVar(&o.WebhookConfigFile, "authorization-webhook-config-file", o.WebhookConfigFile, ""+
		"File with webhook configuration in kubeconfig format, used with --authorization-mode=Webhook. "+
		"The API server will query the remote service to determine access on the API server's secure port.")

	fs.StringVar(&o.WebhookVersion, "authorization-webhook-version", o.WebhookVersion, ""+
		"The API version of the authorization.k8s.io SubjectAccessReview to send to and expect from the webhook.")

	fs.DurationVar(&o.WebhookCacheAuthorizedTTL, "authorization-webhook-cache-authorized-ttl",
		o.WebhookCacheAuthorizedTTL,
		"The duration to cache 'authorized' responses from the webhook authorizer.")

	fs.DurationVar(&o.WebhookCacheUnauthorizedTTL,
		"authorization-webhook-cache-unauthorized-ttl", o.WebhookCacheUnauthorizedTTL,
		"The duration to cache 'unauthorized' responses from the webhook authorizer.")
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
