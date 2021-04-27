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

package json

import (
	"bytes"
	"encoding/json"
	"io"
	"strconv"
	"unicode"
	"unsafe"

	jsoniter "github.com/json-iterator/go"
	"github.com/modern-go/reflect2"
	"sigs.k8s.io/yaml"

	"github.com/yubo/apiserver/staging/runtime"
	"github.com/yubo/apiserver/staging/runtime/serializer/recognizer"
	"github.com/yubo/golib/staging/util/framer"
	utilyaml "github.com/yubo/golib/staging/util/yaml"
	"k8s.io/klog/v2"
)

// NewSerializer creates a JSON serializer that handles encoding versioned objects into the proper JSON form. If typer
// is not nil, the object has the group, version, and kind fields set.
// Deprecated: use NewSerializerWithOptions instead.
func NewSerializer(pretty bool) *Serializer {
	return NewSerializerWithOptions(SerializerOptions{false, pretty, false})
}

// NewYAMLSerializer creates a YAML serializer that handles encoding versioned objects into the proper YAML form. If typer
// is not nil, the object has the group, version, and kind fields set. This serializer supports only the subset of YAML that
// matches JSON, and will error if constructs are used that do not serialize to JSON.
// Deprecated: use NewSerializerWithOptions instead.
func NewYAMLSerializer() *Serializer {
	return NewSerializerWithOptions(SerializerOptions{true, false, false})
}

// NewSerializerWithOptions creates a JSON/YAML serializer that handles encoding versioned objects into the proper JSON/YAML
// form. If typer is not nil, the object has the group, version, and kind fields set. Options are copied into the Serializer
// and are immutable.
func NewSerializerWithOptions(options SerializerOptions) *Serializer {
	return &Serializer{
		//meta: meta,
		//creater:    creater,
		//typer:      typer,
		options:    options,
		identifier: identifier(options),
	}
}

// identifier computes Identifier of Encoder based on the given options.
func identifier(options SerializerOptions) runtime.Identifier {
	result := map[string]string{
		"name":   "json",
		"yaml":   strconv.FormatBool(options.Yaml),
		"pretty": strconv.FormatBool(options.Pretty),
	}
	identifier, err := json.Marshal(result)
	if err != nil {
		klog.Fatalf("Failed marshaling identifier for json Serializer: %v", err)
	}
	return runtime.Identifier(identifier)
}

// SerializerOptions holds the options which are used to configure a JSON/YAML serializer.
// example:
// (1) To configure a JSON serializer, set `Yaml` to `false`.
// (2) To configure a YAML serializer, set `Yaml` to `true`.
// (3) To configure a strict serializer that can return strictDecodingError, set `Strict` to `true`.
type SerializerOptions struct {
	// Yaml: configures the Serializer to work with JSON(false) or YAML(true).
	// When `Yaml` is enabled, this serializer only supports the subset of YAML that
	// matches JSON, and will error if constructs are used that do not serialize to JSON.
	Yaml bool

	// Pretty: configures a JSON enabled Serializer(`Yaml: false`) to produce human-readable output.
	// This option is silently ignored when `Yaml` is `true`.
	Pretty bool

	// Strict: configures the Serializer to return strictDecodingError's when duplicate fields are present decoding JSON or YAML.
	// Note that enabling this option is not as performant as the non-strict variant, and should not be used in fast paths.
	Strict bool
}

// Serializer handles encoding versioned objects into the proper JSON form
type Serializer struct {
	//meta    MetaFactory
	options SerializerOptions
	//creater runtime.ObjectCreater
	//typer   runtime.ObjectTyper

	identifier runtime.Identifier
}

// Serializer implements Serializer
var _ runtime.Serializer = &Serializer{}
var _ recognizer.RecognizingDecoder = &Serializer{}

type customNumberExtension struct {
	jsoniter.DummyExtension
}

func (cne *customNumberExtension) CreateDecoder(typ reflect2.Type) jsoniter.ValDecoder {
	if typ.String() == "interface {}" {
		return customNumberDecoder{}
	}
	return nil
}

type customNumberDecoder struct {
}

func (customNumberDecoder) Decode(ptr unsafe.Pointer, iter *jsoniter.Iterator) {
	switch iter.WhatIsNext() {
	case jsoniter.NumberValue:
		var number jsoniter.Number
		iter.ReadVal(&number)
		i64, err := strconv.ParseInt(string(number), 10, 64)
		if err == nil {
			*(*interface{})(ptr) = i64
			return
		}
		f64, err := strconv.ParseFloat(string(number), 64)
		if err == nil {
			*(*interface{})(ptr) = f64
			return
		}
		iter.ReportError("DecodeNumber", err.Error())
	default:
		*(*interface{})(ptr) = iter.Read()
	}
}

// CaseSensitiveJSONIterator returns a jsoniterator API that's configured to be
// case-sensitive when unmarshalling, and otherwise compatible with
// the encoding/json standard library.
func CaseSensitiveJSONIterator() jsoniter.API {
	config := jsoniter.Config{
		EscapeHTML:             true,
		SortMapKeys:            true,
		ValidateJsonRawMessage: true,
		CaseSensitive:          true,
	}.Froze()
	// Force jsoniter to decode number to interface{} via int64/float64, if possible.
	config.RegisterExtension(&customNumberExtension{})
	return config
}

// StrictCaseSensitiveJSONIterator returns a jsoniterator API that's configured to be
// case-sensitive, but also disallows unknown fields when unmarshalling. It is compatible with
// the encoding/json standard library.
func StrictCaseSensitiveJSONIterator() jsoniter.API {
	config := jsoniter.Config{
		EscapeHTML:             true,
		SortMapKeys:            true,
		ValidateJsonRawMessage: true,
		CaseSensitive:          true,
		DisallowUnknownFields:  true,
	}.Froze()
	// Force jsoniter to decode number to interface{} via int64/float64, if possible.
	config.RegisterExtension(&customNumberExtension{})
	return config
}

// Private copies of jsoniter to try to shield against possible mutations
// from outside. Still does not protect from package level jsoniter.Register*() functions - someone calling them
// in some other library will mess with every usage of the jsoniter library in the whole program.
// See https://github.com/json-iterator/go/issues/265
var caseSensitiveJSONIterator = CaseSensitiveJSONIterator()
var strictCaseSensitiveJSONIterator = StrictCaseSensitiveJSONIterator()

// gvkWithDefaults returns group kind and version defaulting from provided default
//func gvkWithDefaults(actual, defaultGVK schema.GroupVersionKind) schema.GroupVersionKind {
//	if len(actual.Kind) == 0 {
//		actual.Kind = defaultGVK.Kind
//	}
//	if len(actual.Version) == 0 && len(actual.Group) == 0 {
//		actual.Group = defaultGVK.Group
//		actual.Version = defaultGVK.Version
//	}
//	if len(actual.Version) == 0 && actual.Group == defaultGVK.Group {
//		actual.Version = defaultGVK.Version
//	}
//	return actual
//}

// Decode attempts to convert the provided data into YAML or JSON, extract the stored schema kind, apply the provided default gvk, and then
// load that data into an object matching the desired schema kind or the provided into.
// If into is *runtime.Unknown, the raw data will be extracted and no decoding will be performed.
// If into is not registered with the typer, then the object will be straight decoded using normal JSON/YAML unmarshalling.
// If into is provided and the original data is not fully qualified with kind/version/group, the type of the into will be used to alter the returned gvk.
// If into is nil or data's gvk different from into's gvk, it will generate a new Object with ObjectCreater.New(gvk)
// On success or most errors, the method will return the calculated schema kind.
// The gvk calculate priority will be originalData > default gvk > into
func (s *Serializer) Decode(originalData []byte, into runtime.Object) error {
	data := originalData
	if s.options.Yaml {
		altered, err := yaml.YAMLToJSON(data)
		if err != nil {
			return err
		}
		data = altered
	}
	return caseSensitiveJSONIterator.Unmarshal(data, into)
}

// Encode serializes the provided object to the given writer.
func (s *Serializer) Encode(obj runtime.Object, w io.Writer) error {
	if co, ok := obj.(runtime.CacheableObject); ok {
		return co.CacheEncode(s.Identifier(), s.doEncode, w)
	}
	return s.doEncode(obj, w)
}

func (s *Serializer) doEncode(obj runtime.Object, w io.Writer) error {
	if s.options.Yaml {
		json, err := caseSensitiveJSONIterator.Marshal(obj)
		if err != nil {
			return err
		}
		data, err := yaml.JSONToYAML(json)
		if err != nil {
			return err
		}
		_, err = w.Write(data)
		return err
	}

	if s.options.Pretty {
		data, err := caseSensitiveJSONIterator.MarshalIndent(obj, "", "  ")
		if err != nil {
			return err
		}
		_, err = w.Write(data)
		return err
	}
	encoder := json.NewEncoder(w)
	return encoder.Encode(obj)
}

// Identifier implements runtime.Encoder interface.
func (s *Serializer) Identifier() runtime.Identifier {
	return s.identifier
}

// RecognizesData implements the RecognizingDecoder interface.
func (s *Serializer) RecognizesData(data []byte) (ok, unknown bool, err error) {
	if s.options.Yaml {
		// we could potentially look for '---'
		return false, true, nil
	}
	return IsJSONBuffer(data), false, nil
}

// Framer is the default JSON framing behavior, with newlines delimiting individual objects.
var Framer = jsonFramer{}

type jsonFramer struct{}

// NewFrameWriter implements stream framing for this serializer
func (jsonFramer) NewFrameWriter(w io.Writer) io.Writer {
	// we can write JSON objects directly to the writer, because they are self-framing
	return w
}

// NewFrameReader implements stream framing for this serializer
func (jsonFramer) NewFrameReader(r io.ReadCloser) io.ReadCloser {
	// we need to extract the JSON chunks of data to pass to Decode()
	return framer.NewJSONFramedReader(r)
}

// YAMLFramer is the default JSON framing behavior, with newlines delimiting individual objects.
var YAMLFramer = yamlFramer{}

type yamlFramer struct{}

// NewFrameWriter implements stream framing for this serializer
func (yamlFramer) NewFrameWriter(w io.Writer) io.Writer {
	return yamlFrameWriter{w}
}

// NewFrameReader implements stream framing for this serializer
func (yamlFramer) NewFrameReader(r io.ReadCloser) io.ReadCloser {
	// extract the YAML document chunks directly
	return utilyaml.NewDocumentDecoder(r)
}

type yamlFrameWriter struct {
	w io.Writer
}

// Write separates each document with the YAML document separator (`---` followed by line
// break). Writers must write well formed YAML documents (include a final line break).
func (w yamlFrameWriter) Write(data []byte) (n int, err error) {
	if _, err := w.w.Write([]byte("---\n")); err != nil {
		return 0, err
	}
	return w.w.Write(data)
}

// IsJSONBuffer scans the provided buffer, looking
// for an open brace indicating this is JSON.
func IsJSONBuffer(buf []byte) bool {
	return hasJSONPrefix(buf)
}

var jsonPrefix = []byte("{")

// hasJSONPrefix returns true if the provided buffer appears to start with
// a JSON open brace.
func hasJSONPrefix(buf []byte) bool {
	return hasPrefix(buf, jsonPrefix)
}

// Return true if the first non-whitespace bytes in buf is
// prefix.
func hasPrefix(buf []byte, prefix []byte) bool {
	trim := bytes.TrimLeftFunc(buf, unicode.IsSpace)
	return bytes.HasPrefix(trim, prefix)
}
