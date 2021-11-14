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

package reference

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/yubo/golib/api"
	"github.com/yubo/golib/runtime"
)

var (
	// Errors that could be returned by GetReference.
	ErrNilObject = errors.New("can't reference a nil object")
)

// GetReference returns an ObjectReference which refers to the given
// object, or an error if the object doesn't follow the conventions
// that would allow this.
// TODO: should take a meta.Interface see http://issue.k8s.io/7127
func GetReference(obj runtime.Object) (*api.ObjectReference, error) {
	if obj == nil {
		return nil, ErrNilObject
	}
	if ref, ok := obj.(*api.ObjectReference); ok {
		// Don't make a reference to a reference.
		return ref, nil
	}

	return &api.ObjectReference{
		Kind:      getKind(obj),
		Name:      getField("Name", obj),
		Namespace: getField("Namespace", obj),
		UID:       api.UID(getField("UID", obj)),
	}, nil
}

// GetPartialReference is exactly like GetReference, but allows you to set the FieldPath.
func GetPartialReference(obj runtime.Object, fieldPath string) (*api.ObjectReference, error) {
	ref, err := GetReference(obj)
	if err != nil {
		return nil, err
	}
	ref.FieldPath = fieldPath
	return ref, nil
}

func getKind(sample interface{}) string {
	return reflect.Indirect(reflect.ValueOf(sample)).Type().Name()
}

func getField(field string, in interface{}) string {
	if v := reflect.Indirect(reflect.Indirect(
		reflect.ValueOf(in)).FieldByName(field)); v.IsValid() && v.CanInterface() {
		return fmt.Sprintf("%v", v.Interface())
	}
	return ""
}
