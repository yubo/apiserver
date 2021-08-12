/*
Copyright 2014 The Kubernetes Authors.

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

package versioned

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/yubo/apiserver/staging/runtime"
	"github.com/yubo/apiserver/staging/runtime/serializer/streaming"
	"github.com/yubo/apiserver/staging/watch"
)

// Decoder implements the watch.Decoder interface for io.ReadClosers that
// have contents which consist of a series of watchEvent objects encoded
// with the given streaming decoder. The internal objects will be then
// decoded by the embedded decoder.
type Decoder struct {
	objFactory      func() interface{}
	decoder         streaming.Decoder
	embeddedDecoder runtime.Decoder
}

// NewDecoder creates an Decoder for the given writer and codec.
func NewDecoder(obj interface{}, decoder streaming.Decoder, embeddedDecoder runtime.Decoder) *Decoder {
	rt := reflect.Indirect(reflect.ValueOf(obj)).Type()

	return &Decoder{
		decoder:         decoder,
		embeddedDecoder: embeddedDecoder,
		objFactory: func() interface{} {
			return reflect.New(rt).Interface()
		},
	}
}

type WatchEvent struct {
	Type string

	// Object is:
	//  * If Type is Added or Modified: the new state of the object.
	//  * If Type is Deleted: the state of the object immediately before deletion.
	//  * If Type is Error: *Status is recommended; other types may make sense
	//    depending on context.
	Object json.RawMessage
}

// Decode blocks until it can return the next object in the reader. Returns an error
// if the reader is closed or an object can't be decoded.
func (d *Decoder) Decode() (watch.EventType, runtime.Object, error) {
	var got WatchEvent
	res, err := d.decoder.Decode(&got)
	if err != nil {
		return "", nil, err
	}
	if res != &got {
		return "", nil, fmt.Errorf("unable to decode to api.Event")
	}
	switch got.Type {
	case string(watch.Added), string(watch.Modified), string(watch.Deleted), string(watch.Error), string(watch.Bookmark):
	default:
		return "", nil, fmt.Errorf("got invalid watch event type: %v", got.Type)
	}

	obj := d.objFactory()
	_, err = runtime.Decode(d.embeddedDecoder, got.Object, obj)
	return watch.EventType(got.Type), obj, nil
}

// Close closes the underlying r.
func (d *Decoder) Close() {
	d.decoder.Close()
}
