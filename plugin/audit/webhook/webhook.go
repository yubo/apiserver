/*
Copyright 2017 The Kubernetes Authors.

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

// Package webhook implements the audit.Backend interface using HTTP webhooks.
package webhook

import (
	"context"
	"fmt"
	"time"

	auditinternal "github.com/yubo/apiserver/pkg/apis/audit"
	"github.com/yubo/apiserver/pkg/audit"
	"github.com/yubo/apiserver/pkg/util/webhook"
	"github.com/yubo/client-go/rest"
	"github.com/yubo/golib/scheme"
	utilnet "github.com/yubo/golib/util/net"
	utiltrace "github.com/yubo/golib/util/trace"
	"github.com/yubo/golib/util/wait"
	"k8s.io/klog/v2"
)

const (
	// PluginName is the name of this plugin, to be used in help and logs.
	PluginName = "webhook"

	// DefaultInitialBackoffDelay is the default amount of time to wait before
	// retrying sending audit events through a webhook.
	DefaultInitialBackoffDelay = 10 * time.Second
)

// retryOnError enforces the webhook client to retry requests
// on error regardless of its nature.
// The default implementation considers a very limited set of
// 'retriable' errors, assuming correct use of HTTP codes by
// external webhooks.
// That may easily lead to dropped audit events. In fact, there is
// hardly any error that could be a justified reason NOT to retry
// sending audit events if there is even a slight chance that the
// receiving service gets back to normal at some point.
func retryOnError(err error) bool {
	if err != nil {
		return true
	}
	return false
}

func loadWebhook(configFile string, retryBackoff wait.Backoff, customDial utilnet.DialFunc) (*webhook.GenericWebhook, error) {
	w, err := webhook.NewGenericWebhook(scheme.Codec, configFile, retryBackoff, customDial)
	if err != nil {
		return nil, err
	}

	w.ShouldRetry = retryOnError
	return w, nil
}

type backend struct {
	w    *webhook.GenericWebhook
	name string
}

// NewDynamicBackend returns an audit backend configured from a REST client that
// sends events over HTTP to an external service.
func NewDynamicBackend(rc *rest.RESTClient, retryBackoff wait.Backoff) audit.Backend {
	return &backend{
		w: &webhook.GenericWebhook{
			RestClient:   rc,
			RetryBackoff: retryBackoff,
			ShouldRetry:  retryOnError,
		},
		name: fmt.Sprintf("dynamic_%s", PluginName),
	}
}

// NewBackend returns an audit backend that sends events over HTTP to an external service.
func NewBackend(kubeConfigFile string, retryBackoff wait.Backoff, customDial utilnet.DialFunc) (audit.Backend, error) {
	w, err := loadWebhook(kubeConfigFile, retryBackoff, customDial)
	if err != nil {
		return nil, err
	}
	return &backend{w: w, name: PluginName}, nil
}

func (b *backend) Run(stopCh <-chan struct{}) error {
	return nil
}

func (b *backend) Shutdown() {
	// nothing to do here
}

func (b *backend) ProcessEvents(ev ...*auditinternal.Event) bool {
	if err := b.processEvents(ev...); err != nil {
		audit.HandlePluginError(b.String(), err, ev...)
		return false
	}
	return true
}

func (b *backend) processEvents(ev ...*auditinternal.Event) error {
	var list auditinternal.EventList
	for _, e := range ev {
		list.Items = append(list.Items, *e)
	}

	return b.w.WithExponentialBackoff(context.Background(), func() rest.Result {
		trace := utiltrace.New("Call Audit Events webhook",
			utiltrace.Field{Key: "name", Value: b.name},
			utiltrace.Field{Key: "event-count", Value: len(list.Items)})
		// Only log audit webhook traces that exceed a 25ms per object limit plus a 50ms
		// request overhead allowance. The high per object limit used here is primarily to
		// allow enough time for the serialization/deserialization of audit events, which
		// contain nested request and response objects plus additional event fields.
		defer trace.LogIfLong(time.Duration(50+25*len(list.Items)) * time.Millisecond)
		klog.V(10).Infof("event list %+v", list)
		return b.w.RestClient.Post().Body(&list).Do(context.TODO())
	}).Error()
}

func (b *backend) String() string {
	return b.name
}
