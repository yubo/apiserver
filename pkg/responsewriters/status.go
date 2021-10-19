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
	"fmt"
	"net/http"

	"github.com/yubo/golib/api"
	"github.com/yubo/golib/util/runtime"
)

// statusError is an object that can be converted into an api.Status
type statusError interface {
	Status() api.Status
}

// ErrorToAPIStatus converts an error to an api.Status object.
func ErrorToAPIStatus(err error) *api.Status {
	switch t := err.(type) {
	case statusError:
		status := t.Status()
		if len(status.Status) == 0 {
			status.Status = api.StatusFailure
		}
		switch status.Status {
		case api.StatusSuccess:
			if status.Code == 0 {
				status.Code = http.StatusOK
			}
		case api.StatusFailure:
			if status.Code == 0 {
				status.Code = http.StatusInternalServerError
			}
		default:
			runtime.HandleError(fmt.Errorf("apiserver received an error with wrong status field : %#+v", err))
			if status.Code == 0 {
				status.Code = http.StatusInternalServerError
			}
		}
		//status.Kind = "Status"
		//status.APIVersion = "v1"
		//TODO: check for invalid responses
		return &status
	default:
		status := http.StatusInternalServerError
		//TODO: replace me with NewConflictErr
		// Log errors that were not converted to an error status
		// by REST storage - these typically indicate programmer
		// error by not using pkg/api/errors, or unexpected failure
		// cases.
		runtime.HandleError(fmt.Errorf("apiserver received an error that is not an api.Status: %#+v: %v", err, err))
		return &api.Status{
			Status:  api.StatusFailure,
			Code:    int32(status),
			Reason:  api.StatusReasonUnknown,
			Message: err.Error(),
		}
	}
}
