/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package webhook implements a generic HTTP webhook plugin.
package webhook

import (
	"context"
	"fmt"
	"time"

	"github.com/yubo/client-go/rest"
	"github.com/yubo/client-go/tools/clientcmd"
	"github.com/yubo/golib/api"
	"github.com/yubo/golib/api/errors"
	"github.com/yubo/golib/runtime"
	"github.com/yubo/golib/runtime/serializer"
	utilnet "github.com/yubo/golib/util/net"
	"github.com/yubo/golib/util/wait"
)

// defaultRequestTimeout is set for all webhook request. This is the absolute
// timeout of the HTTP request, including reading the response body.
const defaultRequestTimeout = 30 * time.Second

// DefaultRetryBackoffWithInitialDelay returns the default backoff parameters for webhook retry from a given initial delay.
// Handy for the client that provides a custom initial delay only.
func DefaultRetryBackoffWithInitialDelay(initialBackoffDelay time.Duration) wait.Backoff {
	return wait.Backoff{
		Duration: api.Duration{Duration: initialBackoffDelay},
		Factor:   1.5,
		Jitter:   0.2,
		Steps:    5,
	}
}

// GenericWebhook defines a generic client for webhooks with commonly used capabilities,
// such as retry requests.
type GenericWebhook struct {
	RestClient   *rest.RESTClient
	RetryBackoff wait.Backoff
	ShouldRetry  func(error) bool
}

// DefaultShouldRetry is a default implementation for the GenericWebhook ShouldRetry function property.
// If the error reason is one of: networking (connection reset) or http (InternalServerError (500), GatewayTimeout (504), TooManyRequests (429)),
// or apierrors.SuggestsClientDelay() returns true, then the function advises a retry.
// Otherwise it returns false for an immediate fail.
func DefaultShouldRetry(err error) bool {
	// these errors indicate a transient error that should be retried.
	if utilnet.IsConnectionReset(err) || errors.IsInternalError(err) || errors.IsTimeout(err) || errors.IsTooManyRequests(err) {
		return true
	}
	// if the error sends the Retry-After header, we respect it as an explicit confirmation we should retry.
	if _, shouldRetry := errors.SuggestsClientDelay(err); shouldRetry {
		return true
	}
	return false
}

// NewGenericWebhook creates a new GenericWebhook from the provided kubeconfig file.
func NewGenericWebhook(codec runtime.Codec, configFile string, retryBackoff wait.Backoff, customDial utilnet.DialFunc) (*GenericWebhook, error) {
	return newGenericWebhook(codec, configFile, retryBackoff, defaultRequestTimeout, customDial)
}

func newGenericWebhook(codec runtime.Codec, kubeConfigFile string, retryBackoff wait.Backoff, requestTimeout time.Duration, customDial utilnet.DialFunc) (*GenericWebhook, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	loadingRules.ExplicitPath = kubeConfigFile
	loader := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, &clientcmd.ConfigOverrides{})

	clientConfig, err := loader.ClientConfig()
	if err != nil {
		return nil, err
	}

	// Kubeconfigs can't set a timeout, this can only be set through a command line flag.
	//
	// https://github.com/kubernetes/client-go/blob/master/tools/clientcmd/overrides.go
	//
	// Set this to something reasonable so request to webhooks don't hang forever.
	clientConfig.Timeout = requestTimeout

	// Avoid client-side rate limiting talking to the webhook backend.
	// Rate limiting should happen when deciding how many requests to serve.
	clientConfig.QPS = -1

	clientConfig.ContentConfig.NegotiatedSerializer = serializer.NegotiatedSerializerWrapper(
		runtime.SerializerInfo{Serializer: codec},
	)

	clientConfig.Dial = customDial

	restClient, err := rest.UnversionedRESTClientFor(clientConfig)
	if err != nil {
		return nil, err
	}

	return &GenericWebhook{restClient, retryBackoff, DefaultShouldRetry}, nil
}

// WithExponentialBackoff will retry webhookFn() as specified by the given backoff parameters with exponentially
// increasing backoff when it returns an error for which this GenericWebhook's ShouldRetry function returns true,
// confirming it to be retriable. If no ShouldRetry has been defined for the webhook,
// then the default one is used (DefaultShouldRetry).
func (g *GenericWebhook) WithExponentialBackoff(ctx context.Context, webhookFn func() rest.Result) rest.Result {
	var result rest.Result
	shouldRetry := g.ShouldRetry
	if shouldRetry == nil {
		shouldRetry = DefaultShouldRetry
	}
	WithExponentialBackoff(ctx, g.RetryBackoff, func() error {
		result = webhookFn()
		return result.Error()
	}, shouldRetry)
	return result
}

// WithExponentialBackoff will retry webhookFn up to 5 times with exponentially increasing backoff when
// it returns an error for which shouldRetry returns true, confirming it to be retriable.
func WithExponentialBackoff(ctx context.Context, retryBackoff wait.Backoff, webhookFn func() error, shouldRetry func(error) bool) error {
	// having a webhook error allows us to track the last actual webhook error for requests that
	// are later cancelled or time out.
	var webhookErr error
	err := wait.ExponentialBackoffWithContext(ctx, retryBackoff, func() (bool, error) {
		webhookErr = webhookFn()
		if shouldRetry(webhookErr) {
			return false, nil
		}
		if webhookErr != nil {
			return false, webhookErr
		}
		return true, nil
	})

	switch {
	// we check for webhookErr first, if webhookErr is set it's the most important error to return.
	case webhookErr != nil:
		return webhookErr
	case err != nil:
		return fmt.Errorf("webhook call failed: %s", err.Error())
	default:
		return nil
	}
}
