/*
Copyright 2019 The Kubernetes Authors.

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

package runtime_test

import (
	"io"
	"testing"

	"github.com/yubo/apiserver/staging/runtime"
	"github.com/yubo/apiserver/staging/runtime/serializer"
	runtimetesting "github.com/yubo/apiserver/staging/runtime/testing"
)

type mockEncoder struct{}

func (m *mockEncoder) Encode(obj runtime.Object, w io.Writer) error {
	_, err := w.Write([]byte("mock-result"))
	return err
}

func (m *mockEncoder) Identifier() runtime.Identifier {
	return runtime.Identifier("mock-identifier")
}

func TestCacheableObject(t *testing.T) {
	serializer := runtime.NewBase64Serializer(&mockEncoder{}, nil)
	runtimetesting.CacheableObjectTest(t, serializer)
}

func TestCodecs(t *testing.T) {
	test := struct {
		A string
		B int
	}{"hello", 1234}

	codecs := serializer.NewCodecFactory()
	encoder := codecs.LegacyCodec()

	b, err := runtime.Encode(encoder, test)
	if err != nil {
		t.Error(err)
	}

	t.Logf("encode %s", string(b))
}
