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

package runtime

import (
	"fmt"
	"reflect"
)

type notRegisteredErr struct {
	schemeName string
	//gvk        schema.GroupVersionKind
	//target GroupVersioner
	t reflect.Type
}

func NewNotRegisteredErrForKind(schemeName string) error {
	return &notRegisteredErr{schemeName: schemeName}
}

func NewNotRegisteredErrForType(schemeName string, t reflect.Type) error {
	return &notRegisteredErr{schemeName: schemeName, t: t}
}

func NewNotRegisteredErrForTarget(schemeName string, t reflect.Type) error {
	return &notRegisteredErr{schemeName: schemeName, t: t}
}

func NewNotRegisteredGVKErrForTarget(schemeName string) error {
	return &notRegisteredErr{schemeName: schemeName}
}

func (k *notRegisteredErr) Error() string {
	if k.t != nil {
		return fmt.Sprintf("no kind is registered for the type %v in scheme %q", k.t, k.schemeName)
	}

	return fmt.Sprintf("no scheme %q", k.schemeName)
}

// IsNotRegisteredError returns true if the error indicates the provided
// object or input data is not registered.
func IsNotRegisteredError(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(*notRegisteredErr)
	return ok
}

type missingKindErr struct {
	data string
}

func NewMissingKindErr(data string) error {
	return &missingKindErr{data}
}

func (k *missingKindErr) Error() string {
	return fmt.Sprintf("Object 'Kind' is missing in '%s'", k.data)
}

// IsMissingKind returns true if the error indicates that the provided object
// is missing a 'Kind' field.
func IsMissingKind(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(*missingKindErr)
	return ok
}

type missingVersionErr struct {
	data string
}

func NewMissingVersionErr(data string) error {
	return &missingVersionErr{data}
}

func (k *missingVersionErr) Error() string {
	return fmt.Sprintf("Object 'apiVersion' is missing in '%s'", k.data)
}

// IsMissingVersion returns true if the error indicates that the provided object
// is missing a 'Version' field.
func IsMissingVersion(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(*missingVersionErr)
	return ok
}

// strictDecodingError is a base error type that is returned by a strict Decoder such
// as UniversalStrictDecoder.
type strictDecodingError struct {
	message string
	data    string
}

// NewStrictDecodingError creates a new strictDecodingError object.
func NewStrictDecodingError(message string, data string) error {
	return &strictDecodingError{
		message: message,
		data:    data,
	}
}

func (e *strictDecodingError) Error() string {
	return fmt.Sprintf("strict decoder error for %s: %s", e.data, e.message)
}

// IsStrictDecodingError returns true if the error indicates that the provided object
// strictness violations.
func IsStrictDecodingError(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(*strictDecodingError)
	return ok
}
