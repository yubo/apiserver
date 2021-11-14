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

package webhook

import (
	stdjson "encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	api "github.com/yubo/apiserver/pkg/apis/audit"
	clientcmdapi "github.com/yubo/apiserver/tools/clientcmd/api"
	"github.com/yubo/golib/runtime"
	"github.com/yubo/golib/runtime/serializer/json"
	"github.com/yubo/golib/util/wait"
)

// newWebhookHandler returns a handler which receives webhook events and decodes the
// request body. The caller passes a callback which is called on each webhook POST.
// The object passed to cb is of the same type as list.
func newWebhookHandler(t *testing.T, list runtime.Object, cb func(events runtime.Object)) http.Handler {
	s := json.NewSerializer(false)
	return &testWebhookHandler{
		t:          t,
		list:       list,
		onEvents:   cb,
		serializer: s,
	}
}

type testWebhookHandler struct {
	t *testing.T

	list     runtime.Object
	onEvents func(events runtime.Object)

	serializer runtime.Serializer
}

func (t *testWebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	err := func() error {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return fmt.Errorf("read webhook request body: %v", err)
		}

		obj, err := t.serializer.Decode(body, t.list)
		if err != nil {
			return fmt.Errorf("decode request body: %v", err)
		}
		if reflect.TypeOf(obj).Elem() != reflect.TypeOf(t.list).Elem() {
			return fmt.Errorf("expected %T, got %T", t.list, obj)
		}
		t.onEvents(obj)
		return nil
	}()

	if err == nil {
		io.WriteString(w, "{}")
		return
	}
	// In a goroutine, can't call Fatal.
	assert.NoError(t.t, err, "failed to read request body")
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

func newWebhook(t *testing.T, endpoint string) *backend {
	config := clientcmdapi.Config{
		Clusters: map[string]*clientcmdapi.Cluster{
			"": {Server: endpoint, InsecureSkipTLSVerify: true},
		},
	}
	f, err := ioutil.TempFile("", "k8s_audit_webhook_test_")
	require.NoError(t, err, "creating temp file")

	defer func() {
		f.Close()
		os.Remove(f.Name())
	}()

	// NOTE(ericchiang): Do we need to use a proper serializer?
	require.NoError(t, stdjson.NewEncoder(f).Encode(config), "writing kubeconfig")

	retryBackoff := wait.Backoff{
		Duration: 500 * time.Millisecond,
		Factor:   1.5,
		Jitter:   0.2,
		Steps:    5,
	}
	b, err := NewBackend(f.Name(), retryBackoff, nil)
	require.NoError(t, err, "initializing backend")

	return b.(*backend)
}

func TestWebhook(t *testing.T) {
	gotEvents := false

	s := httptest.NewServer(newWebhookHandler(t, &api.EventList{}, func(events runtime.Object) {
		gotEvents = true
	}))
	defer s.Close()

	backend := newWebhook(t, s.URL)

	// Ensure this doesn't return a serialization error.
	event := &api.Event{}
	require.NoError(t, backend.processEvents(event), fmt.Sprintf("failed to send events"))
	require.True(t, gotEvents, fmt.Sprintf("no events received"))
}
