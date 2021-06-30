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

package responsewriters

import (
	stderrs "errors"
	"net/http"
	"reflect"
	"testing"

	"github.com/yubo/golib/api"
	"github.com/yubo/golib/api/errors"
)

func TestBadStatusErrorToAPIStatus(t *testing.T) {
	err := errors.StatusError{}
	actual := ErrorToAPIStatus(&err)
	expected := &api.Status{
		Status: api.StatusFailure,
		Code:   500,
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("%s: Expected %#v, Got %#v", actual, expected, actual)
	}
}

func TestAPIStatus(t *testing.T) {
	cases := map[error]api.Status{
		errors.NewNotFound("bar"): {
			Status:  api.StatusFailure,
			Code:    http.StatusNotFound,
			Reason:  api.StatusReasonNotFound,
			Message: "foos.legacy.kubernetes.io \"bar\" not found",
		},
		errors.NewAlreadyExists("bar"): {
			Status:  api.StatusFailure,
			Code:    http.StatusConflict,
			Reason:  "AlreadyExists",
			Message: "foos \"bar\" already exists",
		},
		errors.NewConflict("bar", stderrs.New("failure")): {
			Status:  api.StatusFailure,
			Code:    http.StatusConflict,
			Reason:  "Conflict",
			Message: "Operation cannot be fulfilled on foos \"bar\": failure",
		},
	}
	for k, v := range cases {
		actual := ErrorToAPIStatus(k)
		if !reflect.DeepEqual(actual, &v) {
			t.Errorf("%s: Expected %#v, Got %#v", k, v, actual)
		}
	}
}
