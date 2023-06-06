package webhook

import (
	"time"

	"github.com/yubo/apiserver/pkg/scheme"
)

// NewWebhook creates a new GenericWebhook from the provided rest.Config.
func NewWebhook(config string, initialBackoffDelay time.Duration) (*GenericWebhook, error) {
	clientConfig, err := LoadKubeconfig(config, nil)
	if err != nil {
		return nil, err
	}

	return NewGenericWebhook(scheme.Codecs, clientConfig, DefaultRetryBackoffWithInitialDelay(initialBackoffDelay))
}
